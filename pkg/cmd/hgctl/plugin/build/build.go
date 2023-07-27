// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	DefaultBuilderRepository = "higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder"
	DefaultBuilderVersion    = "go1.19-tinygo0.25.0-oras1.0.0"

	MediaTypeSpec      = "application/vnd.module.wasm.spec.v1+yaml"
	MediaTypeREADME    = "application/vnd.module.wasm.doc.v1+markdown"
	MediaTypeREADME_ZH = "application/vnd.module.wasm.doc.v1.zh+markdown"
	MediaTypeREADME_EN = "application/vnd.module.wasm.doc.v1.en+markdown"
	MediaTypeIcon      = "application/vnd.module.wasm.icon.v1+png"
	MediaTypePlugin    = "application/vnd.oci.image.layer.v1.tar+gzip"

	HostTempDirPattern = "higress-wasm-go-build-*"

	ContainerWorkDir    = "/workspace"
	ContainerTempDir    = "/higress_temp"
	ContainerOutDir     = "/output"
	ContainerDockerAuth = "/root/.docker/config.json"
)

var (
	optionalProducts = [][2]string{
		{"README_ZH.md", MediaTypeREADME_ZH},
		{"README_EN.md", MediaTypeREADME_EN},
		{"icon.png", MediaTypeIcon},
	}
)

type Builder struct {
	Repository, Version         string
	Input, ProjectName, TempDir string
	Output, OutType, OutDest    string
	Username, Password          string
	OptionFile                  string
	Cmds                        []string
	ContainerConf               types.ContainerCreateConfig
	UID, GID                    string
	DockerAuth                  string
	ModelDir, StructName        string
}

func NewCommand() *cobra.Command {
	var bld Builder

	buildCommand := &cobra.Command{
		Use:     "build",
		Aliases: []string{"bld", "b"},
		Short:   "Build Golang WASM plugin",
		Example: `1. The simplest demo, using "--model(-s)" to specify the WASM plugin configuration structure name, e.g. "BasicAuthConfig"
> hgctl plugin build --model BasicAuthConfig

2. Pushing the products as an OCI image to the specified repository using "--output(-o)"
> docker login
> hgctl plugin build -s "BasicAuthConfig" -o type=image,dest=docker.io/<your_username>/<your_image>
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(bld.build(cmd.OutOrStdout()))
		},
	}

	bld.Repository = DefaultBuilderRepository
	buildCommand.PersistentFlags().StringVarP(&bld.Version, "builder", "t", DefaultBuilderVersion, "The official builder image version")
	buildCommand.PersistentFlags().StringVarP(&bld.Input, "input", "i", "./", "The WASM plugin project directory")
	buildCommand.PersistentFlags().StringVarP(&bld.Output, "output", "o", "type=files,dest=./out", "The output type of build products, which is like `type=[files | image],dest=path`")
	buildCommand.PersistentFlags().StringVarP(&bld.Username, "username", "u", "", "The username for pushing image to the docker repository")
	buildCommand.PersistentFlags().StringVarP(&bld.Password, "password", "p", "", "The password for pushing image to the docker repository")
	buildCommand.PersistentFlags().StringVarP(&bld.OptionFile, "option-file", "f", "./config.yaml", "The options file of building, testing, etc")
	buildCommand.PersistentFlags().StringVarP(&bld.DockerAuth, "docker-auth", "a", "~/.docker/config.json", "The authentication configuration file for pushing image to the docker repository")
	buildCommand.PersistentFlags().StringVarP(&bld.ModelDir, "model-dir", "m", "./", "The directory for the WASM plugin configuration structure")
	buildCommand.PersistentFlags().StringVarP(&bld.StructName, "model", "s", "", "The WASM plugin configuration structure name")

	return buildCommand
}

func (b *Builder) build(w io.Writer) (err error) {
	// 0. check some options
	err = b.checkAndSetOptions()
	if err != nil {
		errors.Wrap(err, "failed to check and set options")
	}

	// 1. generate files `spec.yaml` and `README_{lang}.md` in the temporary directory of host
	tempDir, err := os.MkdirTemp("", HostTempDirPattern)
	if err != nil {
		return errors.Wrap(err, "failed to create host temporary dir")
	}
	defer os.RemoveAll(tempDir)
	b.TempDir = tempDir
	err = b.generateMetadata()
	if err != nil {
		return errors.Wrap(err, "failed to generate metadata")
	}

	// 2. preparing the builder container
	// 2.1. initialize the docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "failed to initialize the docker client")
	}
	defer cli.Close()

	// 2.2. pull the builder image
	builderImage := fmt.Sprintf("%s:%s", b.Repository, b.Version)
	reader, err := cli.ImagePull(ctx, builderImage, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrapf(err, "[×] failed to pull the builder image: %s", builderImage)
	}
	_, err = io.Copy(w, reader)
	if err != nil {
		return errors.Wrap(err, "[×] failed to write the content of the builder image")
	}
	fmt.Fprintf(w, "[√] pull the builder image: %s\n", builderImage)

	// 2.3. set the basic configuration of the builder container
	// TODO: Entrypoint script file will be run inside container instead of executing commands
	b.Cmds = []string{
		"go mod tidy",
		fmt.Sprintf("tinygo build -o %s/plugin.wasm -scheduler=none -target=wasi %s/main.go",
			ContainerTempDir, ContainerWorkDir),
	}
	u, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to get the current user information")
	}
	b.UID, b.GID = u.Uid, u.Gid
	b.DockerAuth, err = getAbsolutePath(b.DockerAuth)
	if err != nil {
		return errors.Wrapf(err, "failed to expand the path of docker authentication configuration")
	}
	b.ContainerConf = types.ContainerCreateConfig{
		Name: "higress-wasm-go-builder",
		Config: &container.Config{
			Image: builderImage,
			Env: []string{
				"GO111MODULE=on",
				"GOPROXY=https://goproxy.cn,direct",
				//fmt.Sprintf("ORAS_USERNAME=%s", b.Username),
				//fmt.Sprintf("ORAS_PASSWORD=%s", b.Password),
			},
			WorkingDir: ContainerWorkDir,
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "host",
			Mounts: []mount.Mount{
				{ // input
					Type:   mount.TypeBind,
					Source: b.Input,
					Target: ContainerWorkDir,
				},
				{ // temp
					Type:   mount.TypeBind,
					Source: b.TempDir,
					Target: ContainerTempDir,
				},
			},
		},
	}

	// 2.4. add additional container configuration for different output type
	switch b.OutType {
	case "files":
		b.filesHandler()
	case "image":
		b.imageHandler()
	default:
		return errors.New("invalid output option, output type is unknown")
	}

	// 3. create and start the builder container to generate the build products
	resp, err := cli.ContainerCreate(ctx, b.ContainerConf.Config, b.ContainerConf.HostConfig, b.ContainerConf.NetworkingConfig, b.ContainerConf.Name)
	if err != nil {
		return errors.Wrap(err, "[×] failed to create the builder container")
	}
	fmt.Fprintf(w, "[√] create container (%s): %s\n", b.ContainerConf.Name, resp.ID)

	defer func() {
		if err != nil {
			fmt.Fprintln(w, fmt.Sprintf("[×] %s", err.Error()))
		}

		err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			fmt.Fprintf(w, "[×] failed to remove container (%s): %s\n", b.ContainerConf.Name, resp.ID)
		}
		fmt.Fprintf(w, "[√] remove container (%s): %s\n", b.ContainerConf.Name, resp.ID)
	}()

	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "[×] failed to start container (%s): %s\n", b.ContainerConf.Name, resp.ID)
	}
	fmt.Fprintf(w, "[√] start container (%s): %s\n", b.ContainerConf.Name, resp.ID)

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}
	_, err = stdcopy.StdCopy(w, w, out)
	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) checkAndSetOptions() error {
	inp, err := getAbsolutePath(b.Input)
	if err != nil {
		return errors.Wrapf(err, "failed to parse input option: %s", b.Input)
	}
	b.Input = inp
	b.ProjectName = filepath.Base(b.Input)

	outParams := strings.Split(b.Output, ",")
	if len(outParams) != 2 {
		return errors.Errorf("invalid output option: %s, must be `type=[files | image],dest=path`", b.Output)
	}
	outTypeKV := strings.Split(outParams[0], "=")
	if len(outTypeKV) != 2 {
		return errors.Errorf("invalid output option: %s, must be `type=[files | image],dest=path`", b.Output)

	}
	b.OutType = strings.TrimSpace(outTypeKV[1])
	outDestKV := strings.Split(outParams[1], "=")
	if len(outDestKV) != 2 {
		return errors.Errorf("invalid output option: %s, must be `type=[files | image],dest=path`", b.Output)
	}
	outDest := strings.TrimSpace(outDestKV[1])
	if b.OutType == "files" {
		outDest, err = getAbsolutePath(strings.TrimSpace(outDestKV[1]))
		if err != nil {
			return errors.Wrapf(err, `failed to parse output destination: "%s"`, outDestKV[1])
		}
		err = os.MkdirAll(outDest, 0755)
		if err != nil && !os.IsExist(err) {
			return errors.Wrapf(err, `failed to mkdir "%s"`, outDest)
		}
	}

	b.OutDest = outDest
	b.Output = fmt.Sprintf("type=%s,dest=%s", b.OutType, b.OutDest)

	return nil
}

func (b *Builder) generateMetadata() error {
	// spec.yaml
	specPath := fmt.Sprintf("%s/spec.yaml", b.TempDir)
	spec, err := os.Create(specPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", specPath)
	}
	defer spec.Close()
	ec := yaml.NewEncoder(spec)
	meta, err := NewWasmPluginMeta(b.ModelDir, b.StructName)
	if err != nil {
		return errors.Wrap(err, "failed to call NewWasmPluginMeta")
	}
	err = ec.Encode(meta)
	if err != nil {
		return errors.Wrap(err, "failed to encode WasmPluginMeta")
	}

	// TODO: More languages need to be supported
	// README_{lang}.md
	// README.md is required, it is zh-CN or en-US version
	usages, err := GetUsageFromMeta(meta)
	if err != nil {
		return errors.Wrap(err, "failed to call GetUsageFromMeta")
	}
	if len(usages) == 0 { // create empty README.md
		mdPath := fmt.Sprintf("%s/README.md", b.TempDir)
		md, err := os.Create(mdPath)
		if err != nil {
			return errors.Wrapf(err, "failed to create %s", mdPath)
		}
		md.Close()

	} else {
		for i, u := range usages {
			var (
				t      *template.Template
				md     *os.File
				mdPath string
			)
			if i == 0 { // default README.md
				mdPath = fmt.Sprintf("%s/README.md", b.TempDir)
			}
			switch u.I18nType {
			case I18nZH_CN:
				t = template.New("MD_zh_CN")
				t = template.Must(t.Parse(MD_zh_CN))
				if i != 0 {
					mdPath = fmt.Sprintf("%s/README_ZH.md", b.TempDir)
				}
			case I18nEN_US:
				t = template.New("MD_en_US")
				t = template.Must(t.Parse(MD_en_US))
				if i != 0 {
					mdPath = fmt.Sprintf("%s/README_EN.md", b.TempDir)
				}
			default:
				t = template.New("MD_zh_CN")
				t = template.Must(t.Parse(MD_zh_CN))
				if i != 0 {
					mdPath = fmt.Sprintf("%s/README_ZH.md", b.TempDir)
				}
			}
			md, err = os.Create(mdPath)
			if err != nil {
				return errors.Wrapf(err, "failed to create %s", mdPath)
			}
			err = t.Execute(md, u)
			if err != nil {
				md.Close()
				return errors.Wrap(err, "failed to execute README.md or README_{lang}.md template")
			}
			md.Close()
		}
	}

	return nil
}

func (b *Builder) filesHandler() {
	b.ContainerConf.HostConfig.Mounts = append(b.ContainerConf.HostConfig.Mounts, mount.Mount{
		// output
		Type:   mount.TypeBind,
		Source: b.OutDest,
		Target: ContainerOutDir,
	})

	// TODO: Entrypoint script file will be run inside container instead of executing commands
	addCmds := []string{
		fmt.Sprintf("mv %s/* %s/", ContainerTempDir, ContainerOutDir),
		fmt.Sprintf("chown -R %s:%s %s/*", b.UID, b.GID, ContainerOutDir),
		"echo '[√] finished building!'",
	}
	b.Cmds = append(b.Cmds, addCmds...)
	b.ContainerConf.Config.Cmd = []string{"bash", "-c", strings.Join(b.Cmds, " && ")}
}

func (b *Builder) imageHandler() {
	products := ""
	for i, p := range optionalProducts {
		fileName := p[0]
		mediaType := p[1]
		if i == 0 {
			products = fmt.Sprintf("%s %s", fileName, mediaType)
		} else {
			products = fmt.Sprintf("%s %s %s", products, fileName, mediaType)
		}
	}

	// TODO: Entrypoint script file will be run inside container instead of executing commands
	// spec.yaml, README.md and plugin.tar.gz are required
	basicCmd := fmt.Sprintf(`cmd="oras push %s -u %s -p %s ./spec.yaml:%s ./README.md:%s"`,
		b.OutDest, b.Username, b.Password, MediaTypeSpec, MediaTypeREADME)

	if b.Username == "" || b.Password == "" {
		basicCmd = fmt.Sprintf(`cmd="oras push %s ./spec.yaml:%s ./README.md:%s"`,
			b.OutDest, MediaTypeSpec, MediaTypeREADME)

		b.ContainerConf.HostConfig.Mounts = append(b.ContainerConf.HostConfig.Mounts, mount.Mount{
			// docker auth
			Type:   mount.TypeBind,
			Source: b.DockerAuth,
			Target: ContainerDockerAuth,
		})
	}

	// append optional files
	ifState := `if [ -e ${f} ]; then cmd="${cmd} ./${f}:${typ}"; fi`
	forCmd := fmt.Sprintf(`products=(%s); for ((i=0; i<${#products[*]}; i=i+2)); do f=${products[i]}; typ=${products[i+1]}; %s; done`, products, ifState)

	addCmds := []string{
		fmt.Sprintf("cd %s", ContainerTempDir),
		"tar czf plugin.tar.gz plugin.wasm",
		basicCmd, // define `cmd`
		forCmd,   // define `products` and append `cmd`
		fmt.Sprintf(`cmd="${cmd} ./plugin.tar.gz:%s"`, MediaTypePlugin), // add plugin type
		"eval ${cmd}", // execute `cmd`
		"echo ${cmd}", // test `cmd`
		"echo '[√] finished building and pushing!'",
	}
	b.Cmds = append(b.Cmds, addCmds...)
	b.ContainerConf.Config.Cmd = []string{"bash", "-c", strings.Join(b.Cmds, " && ")}
}

func getAbsolutePath(path string) (newPath string, err error) {
	if strings.HasPrefix(path, "~") {
		newPath, err = homedir.Expand(path)
		if err != nil {
			return "", errors.Wrapf(err, `failed to expand path: "%s"`, path)
		}
	} else {
		newPath, err = filepath.Abs(path)
		if err != nil {
			return "", errors.Wrapf(err, `failed to get absolute path of "%s"`, path)
		}
	}

	l := len(newPath)
	if l > 1 && newPath[l-1] == '/' { // if l == 1, the path might be "/"
		newPath = newPath[:l-1]
	}

	return newPath, nil
}

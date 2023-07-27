package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	DefaultBuilderRepository = "higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder"
	DefaultBuilderVersion    = "go1.19-tinygo0.25.0-oras1.0.0"

	MediaTypeSpec      = "application/vnd.module.wasm.spec.v1+yaml"
	MediaTypeREADME    = "application/vnd.module.wasm.doc.v1+markdown"
	MediaTypeREADME_EN = "application/vnd.module.wasm.doc.v1.en+markdown"
	MediaTypeIcon      = "application/vnd.module.wasm.icon.v1+png"
	MediaTypePlugin    = "application/vnd.oci.image.layer.v1.tar+gzip"

	HostTempDirPattern = "higress-wasm-go-build-*"

	ContainerWorkDir    = "/workspace"
	ContainerTempDir    = "/higress_temp"
	ContainerOutDir     = "/output"
	ContainerDockerAuth = "/root/.docker/config.json"
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
}

func NewBuildCommand() *cobra.Command {
	var bld Builder

	buildCommand := &cobra.Command{
		Use:     "build",
		Aliases: []string{"bld", "b"},
		Short:   "Build Golang WASM plugin",
		Example: `
     `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(bld.build(cmd.OutOrStdout()))
		},
	}

	bld.Repository = DefaultBuilderRepository
	buildCommand.PersistentFlags().StringVarP(&bld.Version, "builder", "t", DefaultBuilderVersion, "The format is `go${GO_VERSION}-tinygo${TINYGO_VERSION}-oras${ORAS_VERSION}`")
	buildCommand.PersistentFlags().StringVarP(&bld.Input, "input", "i", "./", "")
	buildCommand.PersistentFlags().StringVarP(&bld.Output, "output", "o", "type=files,dest=./out", "type=[files | image],dest=path")
	buildCommand.PersistentFlags().StringVarP(&bld.Username, "username", "u", "", "")
	buildCommand.PersistentFlags().StringVarP(&bld.Password, "password", "p", "", "")
	buildCommand.PersistentFlags().StringVarP(&bld.OptionFile, "option-file", "f", "./config.yaml", "")
	buildCommand.PersistentFlags().StringVarP(&bld.DockerAuth, "docker-auth", "a", "~/.docker/config.json", "")

	return buildCommand
}

func (b *Builder) build(w io.Writer) (err error) {
	// 0.选项校验
	inp, err := getPath(b.Input)
	if err != nil {
		return errors.Wrap(err, "failed to parse input option")
	}
	b.Input = inp
	b.ProjectName = filepath.Base(b.Input)

	outParams := strings.Split(b.Output, ",")
	if len(outParams) != 2 {
		return errors.New("invalid output option: must be `type=[files | image],dest=path`")
	}
	outTypeKV := strings.Split(outParams[0], "=")
	if len(outTypeKV) != 2 {
		return errors.New("invalid output option: must be `type=[files | image],dest=path`")

	}
	b.OutType = strings.TrimSpace(outTypeKV[1])
	outDestKV := strings.Split(outParams[1], "=")
	if len(outDestKV) != 2 {
		return errors.New("invalid output option: must be `type=[files | image],dest=path`")
	}
	outDest, err := getPath(strings.TrimSpace(outDestKV[1]))
	if err != nil {
		return errors.Wrap(err, "failed to parse output path")
	}
	b.OutDest = outDest
	b.Output = fmt.Sprintf("type=%s,dest=%s", b.OutType, b.OutDest)

	// 1. 生成空的 spec.yaml 和 README.md
	tempDir, err := os.MkdirTemp("", HostTempDirPattern)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)
	b.TempDir = tempDir
	err = b.generateMetadata()
	if err != nil {
		return errors.Wrap(err, "failed to generate metadata")
	}

	// 2. 准备 builder 容器
	// 初始化 client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	// 拉取镜像
	builderImage := fmt.Sprintf("%s:%s", b.Repository, b.Version)
	fmt.Fprintf(w, "pull builder image: %s\n", builderImage)
	reader, err := cli.ImagePull(ctx, builderImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}

	// 基本的容器配置
	b.Cmds = []string{
		"go mod tidy",
		fmt.Sprintf("tinygo build -o %s/plugin.wasm -scheduler=none -target=wasi %s/main.go",
			ContainerTempDir, ContainerWorkDir),
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	b.UID, b.GID = u.Uid, u.Gid
	b.DockerAuth, err = homedir.Expand(b.DockerAuth)
	if err != nil {
		return err
	}
	b.ContainerConf = types.ContainerCreateConfig{
		Name: "higress-wasm-go-builder",
		Config: &container.Config{
			Image: builderImage,
			Env: []string{
				"GO111MODULE=on",
				"GOPROXY=https://goproxy.cn,direct",
				fmt.Sprintf("ORAS_USERNAME=%s", b.Username),
				fmt.Sprintf("ORAS_PASSWORD=%s", b.Password),
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

	// 不同输出类型需要添加额外的容器配置
	switch b.OutType {
	case "files":
		b.filesHandler()
	case "image":
		b.imageHandler()
	default:
		return errors.New("invalid output option, output type is unknown")
	}

	// 3. 启动 build 容器，构建 plugin.wasm
	resp, err := cli.ContainerCreate(ctx, b.ContainerConf.Config, b.ContainerConf.HostConfig, b.ContainerConf.NetworkingConfig, b.ContainerConf.Name)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "create container (%s): %s\n", b.ContainerConf.Name, resp.ID)

	// 延迟删除容器
	defer func() {
		if err != nil {
			fmt.Fprintln(w, err.Error())
		}

		fmt.Fprintf(w, "remove container (%s): %s\n", b.ContainerConf.Name, resp.ID)
		err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			fmt.Fprintf(w, "failed to remove container (%s): %s\n", b.ContainerConf.Name, resp.ID)
		}
	}()

	// 启动容器
	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container (%s): %s\n", b.ContainerConf.Name, resp.ID)
	}
	fmt.Fprintf(w, "start container (%s): %s\n", b.ContainerConf.Name, resp.ID)

	// 等待容器结束
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	// 输出容器日志
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

// TODO: Add formal logic
func (b *Builder) generateMetadata() error {
	spec := fmt.Sprintf("%s/spec.yaml", b.TempDir)
	specF, err := os.Create(spec)
	defer specF.Close()
	if err != nil {
		return err
	}

	readme := fmt.Sprintf("%s/README.md", b.TempDir)
	readmeF, err := os.Create(readme)
	defer readmeF.Close()
	if err != nil {
		return err
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

	addCmds := []string{
		fmt.Sprintf("mv %s/* %s/", ContainerTempDir, ContainerOutDir),
		fmt.Sprintf("chown -R %s:%s %s/*", b.UID, b.GID, ContainerOutDir),
		"echo 'finished building!'",
	}
	b.Cmds = append(b.Cmds, addCmds...)
	b.ContainerConf.Config.Cmd = []string{"bash", "-c", strings.Join(b.Cmds, " && ")}
}

func (b *Builder) imageHandler() {
	pushCmd := fmt.Sprintf("oras push %s -u %s -p %s ./spec.yaml:%s ./README.md:%s ./plugin.tar.gz:%s",
		b.OutDest, b.Username, b.Password, MediaTypeSpec, MediaTypeREADME, MediaTypePlugin)
	if b.Username == "" || b.Password == "" {
		pushCmd = fmt.Sprintf("oras push %s ./spec.yaml:%s ./README.md:%s ./plugin.tar.gz:%s",
			b.OutDest, MediaTypeSpec, MediaTypeREADME, MediaTypePlugin)

		b.ContainerConf.HostConfig.Mounts = append(b.ContainerConf.HostConfig.Mounts, mount.Mount{
			// docker auth
			Type:   mount.TypeBind,
			Source: b.DockerAuth,
			Target: ContainerDockerAuth,
		})
	}
	addCmds := []string{
		fmt.Sprintf("cd %s", ContainerTempDir),
		"tar czf plugin.tar.gz plugin.wasm",
		pushCmd,
		"echo 'finished building and pushing!'",
	}
	b.Cmds = append(b.Cmds, addCmds...)
	b.ContainerConf.Config.Cmd = []string{"bash", "-c", strings.Join(b.Cmds, " && ")}
}

// TODO: reuse filepath.Abs to fix this func
// 获取绝对路径，并且当 path 不是根目录 '/' 时，返回的 dir 不以 '/' 结尾
func getPath(path string) (dir string, err error) {
	l := len(path)
	dir = path
	if l == 0 {
		return "", errors.New("invalid path")
	}

	// ./ -> .
	// ./aa/bb/ -> ./aa/bb
	// /aa/bb/ -> /aa/bb
	// / -> /
	if l > 1 && dir[l-1] == '/' {
		dir = dir[:l-1]
		l -= 1
	}

	// . -> /to/path
	// ./aa/bb -> /to/path/aa/bb
	// /aa/bb -> /aa/bb
	// / -> /
	if l > 0 && dir[0] == '.' {
		left, err := os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, "failed to call os.Getwd()")
		}
		right := ""
		if l > 2 {
			right = dir[2:]
		}
		if right == "" {
			dir = left
		} else {
			dir = fmt.Sprintf("%s/%s", left, right)
		}
	}
	return dir, nil
}

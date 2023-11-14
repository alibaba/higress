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
	"os"
	"text/template"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"
)

const (
	filesDockerEntrypoint = `#!/bin/bash
set -e
{{- if eq .Debug true }}
set -x
{{- end }}

go mod tidy
tinygo build -o {{ .BuildDestDir }}/plugin.wasm -scheduler=none -gc=custom -tags='custommalloc nottinygc_finalizer' -target=wasi {{ .BuildSrcDir }}

mv {{ .BuildDestDir }}/* {{ .Output }}/
chown -R {{ .UID }}:{{ .GID }} {{ .Output }}
`
	imageDockerEntrypoint = `#!/bin/bash
set -e
{{- if eq .Debug true }}
set -x
{{- end }}

go mod tidy
tinygo build -o {{ .BuildDestDir }}/plugin.wasm -scheduler=none -gc=custom -tags='custommalloc nottinygc_finalizer' -target=wasi {{ .BuildSrcDir }}

cd {{ .BuildDestDir }}
tar czf plugin.tar.gz plugin.wasm
cmd="{{ .BasicCmd }}"
products=({{ .Products }})
for ((i=0; i<${#products[*]}; i=i+2)); do 
  f=${products[i]}
  typ=${products[i+1]}
  if [ -e ${f} ]; then 
    cmd="${cmd} ./${f}:${typ}" 
  fi
done
cmd="${cmd} ./plugin.tar.gz:{{ .MediaTypePlugin }}"
eval ${cmd}
`
)

type FilesTmplFields struct {
	BuildSrcDir  string
	BuildDestDir string
	Output       string
	UID, GID     string
	Debug        bool
}

type ImageTmplFields struct {
	BuildSrcDir        string
	BuildDestDir       string
	Output             string
	Username, Password string
	BasicCmd           string
	Products           string
	MediaTypePlugin    string
	Debug              bool
}

func genFilesDockerEntrypoint(ft *FilesTmplFields, target string) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = template.Must(template.New("FilesDockerEntrypoint").Parse(filesDockerEntrypoint)).Execute(f, ft); err != nil {
		return err
	}

	return nil
}

func genImageDockerEntrypoint(it *ImageTmplFields, target string) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = template.Must(template.New("ImageDockerEntrypoint").Parse(imageDockerEntrypoint)).Execute(f, it); err != nil {
		return err
	}

	return nil
}

const (
	readme_zh_CN = `> 该插件用法文件根据源代码自动生成，请根据需求自行修改！

# 功能说明

{{ .Description }}

# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
{{- range .ConfigEntries }}
| {{ .Name }} | {{ .Type }} | {{ .Requirement }} | {{ .Default }} | {{ .Description }} |
{{- end }}

# 配置示例

` + "```yaml" + `
{{ .Example }}
` + "```" + `
`

	readme_en_US = `> THIS PLUGIN USAGE FILE IS AUTOMATICALLY GENERATED BASED ON THE SOURCE CODE. MODIFY IT AS REQUIRED!

# Description

{{ .Description }}

# Configuration

| Name | Type | Requirement |  Default | Description |
| -------- | -------- | -------- | -------- | -------- |
{{- range .ConfigEntries }}
| {{ .Name }} | {{ .Type }} | {{ .Requirement }} | {{ .Default }} | {{ .Description }} |
{{- end }}

# Examples

` + "```yaml" + `
{{ .Example }}
` + "```" + `
`
)

func genMarkdownUsage(u *types.WasmUsage, dir string, suffix bool) error {
	md, err := os.Create(i18n2MDTitle(u.I18nType, dir, suffix))
	if err != nil {
		return err
	}
	defer md.Close()

	if err = template.Must(template.New("MD_Usage").Parse(i18n2MD(u.I18nType))).Execute(md, u); err != nil {
		return err
	}

	return nil
}

func i18n2MD(i18n types.I18nType) string {
	switch i18n {
	case types.I18nEN_US:
		return readme_en_US
	case types.I18nZH_CN:
		return readme_zh_CN
	default:
		return readme_zh_CN
	}
}

func i18n2MDTitle(i18n types.I18nType, dir string, suffix bool) string {
	var file string
	if !suffix {
		file = "README.md"
	} else {
		switch i18n {
		case types.I18nEN_US:
			file = "README_EN.md"
		case types.I18nZH_CN:
			file = "README_ZH.md"
		default:
			file = "README_ZH.md"
		}
	}

	return dir + "/" + file
}

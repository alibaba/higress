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

var (
	MD_zh_CN = `> 该插件用法文件根据源代码自动生成，请根据需求自行修改！

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

	MD_en_US = `> THIS PLUGIN USAGE FILE IS AUTOMATICALLY GENERATED BASED ON THE SOURCE CODE. MODIFY IT AS REQUIRED!

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

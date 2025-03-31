// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tidwall/sjson"
)

func TestRestToolConfig(t *testing.T) {
	// Example REST tool configuration
	configJSON := `
{
  "server": {
    "name": "rest-amap-server",
    "config": {
      "apiKey": "xxxxx"
    }
  },
  "tools": [
    {
      "name": "maps-geo",
      "description": "将详细的结构化地址转换为经纬度坐标。支持对地标性名胜景区、建筑物名称解析为经纬度坐标",
      "args": [
        {
          "name": "address",
          "description": "待解析的结构化地址信息",
          "required": true
        },
        {
          "name": "city",
          "description": "指定查询的城市",
          "required": false
        }
      ],
      "requestTemplate": {
        "url": "https://restapi.amap.com/v3/geocode/geo?key={{.config.apiKey}}&address={{.args.address}}&city={{.args.city}}&source=ts_mcp",
        "method": "GET",
        "headers": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ]
      },
      "responseTemplate": {
        "body": "# 地理编码信息\n{{- range $index, $geo := .Geocodes }}\n## 地点 {{add $index 1}}\n\n- **国家**: {{ $geo.Country }}\n- **省份**: {{ $geo.Province }}\n- **城市**: {{ $geo.City }}\n- **城市代码**: {{ $geo.Citycode }}\n- **区/县**: {{ $geo.District }}\n- **街道**: {{ $geo.Street }}\n- **门牌号**: {{ $geo.Number }}\n- **行政编码**: {{ $geo.Adcode }}\n- **坐标**: {{ $geo.Location }}\n- **级别**: {{ $geo.Level }}\n{{- end }}"
      }
    }
  ]
}
`

	// Parse the config to verify it's valid JSON
	var configData map[string]interface{}
	err := json.Unmarshal([]byte(configJSON), &configData)
	if err != nil {
		t.Fatalf("Invalid JSON config: %v", err)
	}

	// Example tool configuration
	tool := RestTool{
		Name:        "maps-geo",
		Description: "将详细的结构化地址转换为经纬度坐标。支持对地标性名胜景区、建筑物名称解析为经纬度坐标",
		Args: []RestToolArg{
			{
				Name:        "address",
				Description: "待解析的结构化地址信息",
				Required:    true,
			},
			{
				Name:        "city",
				Description: "指定查询的城市",
				Required:    false,
			},
		},
		RequestTemplate: RestToolRequestTemplate{
			URL:    "https://restapi.amap.com/v3/geocode/geo?key={{.config.apiKey}}&address={{.args.address}}&city={{.args.city}}&source=ts_mcp",
			Method: "GET",
			Headers: []RestToolHeader{
				{
					Key:   "Content-Type",
					Value: "application/json",
				},
			},
		},
		ResponseTemplate: RestToolResponseTemplate{
			Body: `# 地理编码信息
{{- range $index, $geo := .Geocodes }}
## 地点 {{add $index 1}}

- **国家**: {{ $geo.Country }}
- **省份**: {{ $geo.Province }}
- **城市**: {{ $geo.City }}
- **城市代码**: {{ $geo.Citycode }}
- **区/县**: {{ $geo.District }}
- **街道**: {{ $geo.Street }}
- **门牌号**: {{ $geo.Number }}
- **行政编码**: {{ $geo.Adcode }}
- **坐标**: {{ $geo.Location }}
- **级别**: {{ $geo.Level }}
{{- end }}`,
		},
	}

	// Parse templates
	err = tool.parseTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var templateData []byte
	templateData, _ = sjson.SetBytes(templateData, "config", map[string]interface{}{"apiKey": "test-api-key"})
	templateData, _ = sjson.SetBytes(templateData, "args", map[string]interface{}{"address": "北京市朝阳区阜通东大街6号", "city": "北京"})

	// Test URL template
	url, err := executeTemplate(tool.parsedURLTemplate, templateData)
	if err != nil {
		t.Fatalf("Failed to execute URL template: %v", err)
	}

	expectedURL := "https://restapi.amap.com/v3/geocode/geo?key=test-api-key&address=北京市朝阳区阜通东大街6号&city=北京&source=ts_mcp"
	if url != expectedURL {
		t.Errorf("URL template rendering failed. Expected: %s, Got: %s", expectedURL, url)
	}

	// Test response template with sample data
	sampleResponse := `
		{"Geocodes": [
			{
				"Country":  "中国",
				"Province": "北京市",
				"City":     "北京市",
				"Citycode": "010",
				"District": "朝阳区",
				"Street":   "阜通东大街",
				"Number":   "6号",
				"Adcode":   "110105",
				"Location": "116.483038,39.990633",
				"Level":    "门牌号",
			}]}`

	result, err := executeTemplate(tool.parsedResponseTemplate, []byte(sampleResponse))
	if err != nil {
		t.Fatalf("Failed to execute response template: %v", err)
	}

	// Just check that the result contains expected substrings
	expectedSubstrings := []string{
		"# 地理编码信息",
		"## 地点 1",
		"**国家**: 中国",
		"**省份**: 北京市",
		"**坐标**: 116.483038,39.990633",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(result, substr) {
			t.Errorf("Response template rendering failed. Expected substring not found: %s", substr)
		}
	}
}

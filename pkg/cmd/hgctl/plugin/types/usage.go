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

package types

import (
	"fmt"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"
)

type WasmUsage struct {
	I18nType      I18nType
	Description   string
	ConfigEntries []ConfigEntry
	Example       string
}

type ConfigEntry struct {
	Name        string
	Type        string
	Requirement string
	Default     string
	Description string
}

func GetUsageFromMeta(meta *WasmPluginMeta) ([]*WasmUsage, error) {
	usages := make([]*WasmUsage, 0)
	example := ""
	schema := meta.Spec.ConfigSchema.OpenAPIV3Schema
	if schema != nil && schema.Example != nil && len(schema.Example.Raw) > 0 {
		example = string(schema.Example.Raw)
	}

	m := utils.NewOrderedMap()
	for i18n, desc := range meta.Info.XDescriptionI18n {
		m.Set(string(i18n), desc)
	}
	for i18n := range meta.Info.XTitleI18n {
		if _, ok := m.Get(string(i18n)); !ok {
			m.Set(string(i18n), "No description")
		}
	}
	if len(m.Keys()) == 0 {
		m.Set(string(I18nEN_US), "No description")
	}
	for _, i18n := range m.Keys() {
		desc, ok := m.Get(i18n)
		if !ok {
			continue
		}

		u := WasmUsage{
			I18nType:      I18nType(i18n),
			Description:   desc.(string),
			ConfigEntries: make([]ConfigEntry, 0),
			Example:       example,
		}
		getConfigEntryFromSchema(schema, &u.ConfigEntries, "", "", I18nType(i18n), false)
		usages = append(usages, &u)
	}

	return usages, nil
}

func getConfigEntryFromSchema(schema *JSONSchemaProps, entries *[]ConfigEntry, parent, name string, i18n I18nType, required bool) {
	newName := constructName(parent, name)
	reqs := schema.HandleRequirements(required)

	switch schema.Type {
	case "object":
		for _, name := range schema.Properties.Keys() {
			val, ok := schema.Properties.Get(name)
			if !ok {
				continue
			}
			props := val.(JSONSchemaProps)
			required = schema.IsRequired(name)
			getConfigEntryFromSchema(&props, entries, newName, name, i18n, required)
		}
	case "array":
		itemType := schema.Items.Schema.Type
		e := ConfigEntry{
			Name:        newName,
			Type:        fmt.Sprintf("array of %s", itemType),
			Requirement: RequirementsJoinByI18n(reqs, i18n),
			Default:     schema.GetDefaultValue(),
			Description: schema.XDescriptionI18n[i18n],
		}
		*entries = append(*entries, e)
		if itemType == "object" {
			getConfigEntryFromSchema(schema.Items.Schema, entries, fmt.Sprintf("%s[*]", newName), "", i18n, false)
		}
	default:
		e := ConfigEntry{
			Name:        newName,
			Type:        schema.Type,
			Requirement: RequirementsJoinByI18n(reqs, i18n),
			Default:     schema.GetDefaultValue(),
			Description: schema.XDescriptionI18n[i18n],
		}
		*entries = append(*entries, e)
	}
}

func constructName(parent, name string) string {
	newName := name
	if parent != "" {
		if name != "" {
			newName = fmt.Sprintf("%s.%s", parent, name)
		} else {
			newName = parent
		}
	}
	return newName
}

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
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/orderedmap"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// WasmPluginMeta is used to describe WASM plugin metadata,
// see https://higress.io/en-us/docs/user/wasm-image-spec/
type WasmPluginMeta struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Info       WasmPluginInfo `json:"info" yaml:"info"`
	Spec       WasmPluginSpec `json:"spec" yaml:"spec"`
}

func defaultWasmPluginMeta() *WasmPluginMeta {
	return &WasmPluginMeta{
		APIVersion: "1.0.0",
		Info: WasmPluginInfo{
			Category:         CategoryCustom,
			Name:             "Unnamed",
			XTitleI18n:       make(map[I18nType]string),
			XDescriptionI18n: make(map[I18nType]string),
			Version:          "0.1.0",
		},
		Spec: WasmPluginSpec{
			Phase:    PhaseUnspecified,
			Priority: 0,
		},
	}
}

// ParseSpecYAML parses the `spec.yaml` to WasmPluginMeta
func ParseSpecYAML(spec string) (*WasmPluginMeta, error) {
	f, err := os.Open(spec)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m WasmPluginMeta
	dc := k8syaml.NewYAMLOrJSONDecoder(f, 4096)
	if err = dc.Decode(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// ParseGoSrc parses the config model of the golang WASM plugin project to WasmPluginMeta
func ParseGoSrc(dir, model string) (*WasmPluginMeta, error) {
	mp, err := NewModelParser(dir)
	if err != nil {
		return nil, err
	}
	m, err := mp.GetModel(model)
	if err != nil {
		return nil, err
	}
	meta := defaultWasmPluginMeta()
	meta.setByConfigModel(m)
	return meta, nil
}

func (meta *WasmPluginMeta) setByConfigModel(model *Model) {
	_, schema := recursiveSetSchema(model, nil)
	meta.Spec.ConfigSchema.OpenAPIV3Schema = schema
	meta.setModelAnnotations(model.Doc)
}

func recursiveSetSchema(model *Model, parent *JSONSchemaProps) (string, *JSONSchemaProps) {
	cur := NewJSONSchemaProps()
	cur.Type = model.Type
	if parent != nil {
		cur.HandleFieldAnnotations(model.Doc)
	}
	newName := cur.HandleFieldTags(model.Tag, parent, model.Name)
	if IsArray(model.Type) {
		cur.Type = "array"
		itemModel := &*model
		itemModel.Type = GetItemType(model.Type)
		_, itemSchema := recursiveSetSchema(itemModel, nil)
		cur.Items = &JSONSchemaPropsOrArray{Schema: itemSchema}
	} else if IsMap(model.Type) {
		cur.Type = "object"
		valueModel := &*model
		valueModel.Type = GetValueType(model.Type)
		valueModel.Tag = ""
		valueModel.Doc = ""
		_, valueSchema := recursiveSetSchema(valueModel, nil)
		cur.AdditionalProperties = &JSONSchemaPropsOrBool{Schema: valueSchema}
	} else if IsObject(model.Type) { // type may be `array of object`, and it is handled in the first branch
		cur.Properties = make(map[string]JSONSchemaProps)
		recursiveObjectProperties(cur, model)
	}
	return newName, cur
}

func recursiveObjectProperties(parent *JSONSchemaProps, model *Model) {
	for _, field := range model.Fields {
		name, child := recursiveSetSchema(&field, parent)
		parent.Properties[name] = *child
	}
}

func (meta *WasmPluginMeta) setModelAnnotations(comment string) {
	as := GetAnnotations(comment)
	for _, a := range as {
		switch a.Type {
		// Info
		case ACategory:
			meta.Info.Category = Category(a.Text)
		case AName:
			meta.Info.Name = a.Text
		case ATitle:
			if meta.Info.Title == "" {
				meta.Info.Title = a.Text
			}
			meta.Info.XTitleI18n[a.I18nType] = a.Text
		case ADescription:
			if meta.Info.Description == "" {
				meta.Info.Description = a.Text
			}
			meta.Info.XDescriptionI18n[a.I18nType] = a.Text
		case AIconUrl:
			meta.Info.IconUrl = a.Text
		case AVersion:
			meta.Info.Version = a.Text
		case AContactName:
			meta.Info.Contact.Name = a.Text
		case AContactUrl:
			meta.Info.Contact.Url = a.Text
		case AContactEmail:
			meta.Info.Contact.Email = a.Text

		// Spec
		case APhase:
			meta.Spec.Phase = Phase(a.Text)
		case APriority:
			priority, err := strconv.ParseInt(a.Text, 10, 64)
			if err != nil {
				priority = 0
			}
			meta.Spec.Priority = priority

		// Schema
		case AExample:
			meta.Spec.ConfigSchema.OpenAPIV3Schema.Example = &JSON{Raw: []byte(a.Text)}
		case AScope:
			meta.Spec.ConfigSchema.OpenAPIV3Schema.Scope = Scope(a.Text)
		}
	}
}

type WasmPluginInfo struct {
	Category         Category            `json:"category" yaml:"category"`
	Name             string              `json:"name" yaml:"name"`
	Title            string              `json:"title,omitempty" yaml:"title,omitempty"`
	XTitleI18n       map[I18nType]string `json:"x-title-i18n,omitempty" yaml:"x-title-i18n,omitempty"`
	Description      string              `json:"description,omitempty" yaml:"description,omitempty"`
	XDescriptionI18n map[I18nType]string `json:"x-description-i18n,omitempty" yaml:"x-description-i18n,omitempty"`
	IconUrl          string              `json:"iconUrl,omitempty" yaml:"iconUrl,omitempty"`
	Version          string              `json:"version" yaml:"version"`
	Contact          Contact             `json:"contact,omitempty" yaml:"contact,omitempty"`
}

type Category string

const (
	CategoryAuth        Category = "auth"
	CategorySecurity    Category = "security"
	CategoryProtocol    Category = "protocol"
	CategoryFlowControl Category = "flow-control"
	CategoryFlowMonitor Category = "flow-monitor"
	CategoryCustom      Category = "custom"
	CategoryDefault              = CategoryCustom
)

const (
	IconAuth        = "https://img.alicdn.com/imgextra/i4/O1CN01BPFGlT1pGZ2VDLgaH_!!6000000005333-2-tps-42-42.png"
	IconSecurity    = "https://img.alicdn.com/imgextra/i1/O1CN01jKT9vC1O059vNaq5u_!!6000000001642-2-tps-42-42.png"
	IconProtocol    = "https://img.alicdn.com/imgextra/i2/O1CN01xIywow1mVGuRUjbhe_!!6000000004959-2-tps-42-42.png"
	IconFlowControl = "https://img.alicdn.com/imgextra/i3/O1CN01bAFa9k1t1gdQcVTH0_!!6000000005842-2-tps-42-42.png"
	IconFlowMonitor = "https://img.alicdn.com/imgextra/i4/O1CN01aet3s61MoLOEEhRIo_!!6000000001481-2-tps-42-42.png"
	IconCustom      = "https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png"
	IconDefault     = IconCustom
)

func Category2IconUrl(category Category) string {
	switch category {
	case CategoryAuth:
		return IconAuth
	case CategorySecurity:
		return IconSecurity
	case CategoryProtocol:
		return IconProtocol
	case CategoryFlowControl:
		return IconFlowControl
	case CategoryFlowMonitor:
		return IconFlowMonitor
	case CategoryCustom:
		return IconCustom
	default:
		return IconDefault
	}
}

type I18nType string

const (
	I18nZH_CN     I18nType = "zh-CN" // default
	I18nEN_US     I18nType = "en-US"
	I18nUndefined I18nType = "undefined" // i18n type is empty in the annotation
	I18nUnknown   I18nType = "unknown"
	I18nDefault            = I18nEN_US
)

func str2I18nType(typ string) I18nType {
	switch strings.ToLower(typ) {
	case "zh-cn":
		return I18nZH_CN
	case "en-us":
		return I18nEN_US
	default:
		return I18nUnknown
	}
}

type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Url   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type WasmPluginSpec struct {
	// Phase refers to https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/#PluginPhase
	Phase Phase `json:"phase" yaml:"phase"`

	// Priority refers to https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/#WasmPlugin
	Priority int64 `json:"priority" yaml:"priority"`

	ConfigSchema ConfigSchema `json:"configSchema" yaml:"configSchema"`
}

type Phase string

const (
	PhaseUnspecified Phase = "UNSPECIFIED_PHASE"
	PhaseAuthn       Phase = "AUTHN"
	PhaseAuthz       Phase = "AUTHZ"
	PhaseStats       Phase = "STATS"
	PhaseDefault           = PhaseUnspecified
)

type ConfigSchema struct {
	OpenAPIV3Schema *JSONSchemaProps `json:"openAPIV3Schema" yaml:"openAPIV3Schema"`
}

// GetConfigExample returns a pretty WASM plugin config example
func (meta *WasmPluginMeta) GetConfigExample() string {
	s := meta.Spec.ConfigSchema.OpenAPIV3Schema
	if s != nil {
		return s.GetExample()
	}
	return ""
}

// getLanguageUnionOrderMap returns a ordered map of language union of title and description.
// If there is a language type in title that description does not have, the value is "No description"
func (meta *WasmPluginMeta) getLanguageUnionOrderMap() *orderedmap.OrderedMap {
	m := orderedmap.New()
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
	m.SortKeys(sort.Strings)
	return m
}

// WasmUsage is used to describe WASM plugin usage in the Markdown document
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

// GetUsages returns WASM plugin usages in different languages
func (meta *WasmPluginMeta) GetUsages() ([]WasmUsage, error) {
	usages := make([]WasmUsage, 0)
	example := meta.GetConfigExample()
	m := meta.getLanguageUnionOrderMap()
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
		getConfigEntries(meta.Spec.ConfigSchema.OpenAPIV3Schema, &u.ConfigEntries, I18nType(i18n))
		usages = append(usages, u)
	}

	return usages, nil
}

func getConfigEntries(schema *JSONSchemaProps, entries *[]ConfigEntry, i18n I18nType) {
	doGetConfigEntries(schema, entries, "", "", i18n, false)
}

func doGetConfigEntries(schema *JSONSchemaProps, entries *[]ConfigEntry, parentName, name string, i18n I18nType, required bool) {
	newName := constructName(parentName, name)
	switch schema.Type {
	case "object":
		m := schema.GetPropertiesOrderMap()
		for _, fieldName := range m.Keys() {
			val, ok := m.Get(fieldName)
			if !ok {
				continue
			}
			props := val.(JSONSchemaProps)
			required = schema.IsRequired(fieldName)
			doGetConfigEntries(&props, entries, newName, fieldName, i18n, required)
		}
	case "array":
		itemType := schema.Items.Schema.Type
		e := ConfigEntry{
			Name:        newName,
			Type:        ArrayPrefix + itemType,
			Requirement: schema.JoinRequirementsBy(i18n, required),
			Default:     schema.GetDefaultValue(),
			Description: schema.XDescriptionI18n[i18n],
		}
		*entries = append(*entries, e)
		if itemType == "object" {
			doGetConfigEntries(schema.Items.Schema, entries, newName+"[*]", "", i18n, false)
		}
	default:
		e := ConfigEntry{
			Name:        newName,
			Type:        schema.Type,
			Requirement: schema.JoinRequirementsBy(i18n, required),
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

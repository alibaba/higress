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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/fatih/structtag"
	"github.com/iancoleman/orderedmap"
)

// JSONSchemaProps is a JSON-Schema following Specification Draft 4 (http://json-schema.org/).
// Borrowed from https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/v1/types_jsonschema.go
type JSONSchemaProps struct {
	ID                   string                     `json:"id,omitempty" yaml:"id,omitempty"`
	Schema               JSONSchemaURL              `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Ref                  *string                    `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type                 string                     `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string                     `json:"format,omitempty" yaml:"format,omitempty"`
	Scope                Scope                      `json:"scope,omitempty" yaml:"scope,omitempty"`
	Title                string                     `json:"title,omitempty" yaml:"title,omitempty"`
	XTitleI18n           map[I18nType]string        `json:"x-title-i18n,omitempty" yaml:"x-title-i18n,omitempty"`
	Description          string                     `json:"description,omitempty" yaml:"description,omitempty"`
	XDescriptionI18n     map[I18nType]string        `json:"x-description-i18n,omitempty" yaml:"x-description-i18n,omitempty"`
	Default              *JSON                      `json:"default,omitempty" yaml:"default,omitempty"`
	Minimum              *float64                   `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     bool                       `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	Maximum              *float64                   `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     bool                       `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	MinLength            *int64                     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength            *int64                     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern              string                     `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems             *int64                     `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems             *int64                     `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems          bool                       `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MultipleOf           *float64                   `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Enum                 []JSON                     `json:"enum,omitempty" yaml:"enum,omitempty"`
	MinProperties        *int64                     `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties        *int64                     `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Required             []string                   `json:"required,omitempty" yaml:"required,omitempty"`
	Items                *JSONSchemaPropsOrArray    `json:"items,omitempty" yaml:"items,omitempty"`
	AllOf                []JSONSchemaProps          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf                []JSONSchemaProps          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf                []JSONSchemaProps          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Not                  *JSONSchemaProps           `json:"not,omitempty" yaml:"not,omitempty"`
	Properties           map[string]JSONSchemaProps `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *JSONSchemaPropsOrBool     `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	PatternProperties    map[string]JSONSchemaProps `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	Dependencies         JSONSchemaDependencies     `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	AdditionalItems      *JSONSchemaPropsOrBool     `json:"additionalItems,omitempty" yaml:"additionalItems,omitempty"`
	Definitions          JSONSchemaDefinitions      `json:"definitions,omitempty" yaml:"definitions,omitempty"`
	ExternalDocs         *ExternalDocumentation     `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Example              *JSON                      `json:"example,omitempty" yaml:"example,omitempty"`
	Nullable             bool                       `json:"nullable,omitempty" yaml:"nullable,omitempty"`
}

type Scope string

const (
	ScopeGlobal   Scope = "GLOBAL"
	ScopeInstance Scope = "INSTANCE"
	ScopeAll      Scope = "ALL"
	ScopeDefault        = ScopeInstance
)

// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
type JSON struct {
	Raw []byte `json:"-" yaml:"-"`
}

// JSONSchemaPropsOrArray represents a value that can either be a JSONSchemaProps
// or an array of JSONSchemaProps. Mainly here for serialization purposes.
type JSONSchemaPropsOrArray struct {
	Schema      *JSONSchemaProps
	JSONSchemas []JSONSchemaProps
}

// JSONSchemaPropsOrBool represents JSONSchemaProps or a boolean value.
// Defaults to true for the boolean property.
type JSONSchemaPropsOrBool struct {
	Allows bool
	Schema *JSONSchemaProps
}

// JSONSchemaDependencies represent a dependencies property.
type JSONSchemaDependencies map[string]JSONSchemaPropsOrStringArray

// JSONSchemaPropsOrStringArray represents a JSONSchemaProps or a string array.
type JSONSchemaPropsOrStringArray struct {
	Schema   *JSONSchemaProps
	Property []string
}

// JSONSchemaURL represents a schema url.
type JSONSchemaURL string

// JSONSchemaDefinitions contains the models explicitly defined in this spec.
type JSONSchemaDefinitions map[string]JSONSchemaProps

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

func NewJSONSchemaProps() *JSONSchemaProps {
	return &JSONSchemaProps{
		XTitleI18n:       make(map[I18nType]string),
		XDescriptionI18n: make(map[I18nType]string),
		Properties:       make(map[string]JSONSchemaProps),
	}
}

// IsRequired determines whether the given `name` field is required
func (s *JSONSchemaProps) IsRequired(name string) bool {
	req := false
	for _, n := range s.Required {
		if name == n {
			req = true
			break
		}
	}
	return req
}

// GetDefaultValue returns the default value of the schema
func (s *JSONSchemaProps) GetDefaultValue() string {
	d := "-"
	if s.Default == nil {
		return d
	}
	if len(s.Default.Raw) > 0 {
		d = string(s.Default.Raw)
	}
	return d
}

// GetExample returns the pretty example of the schema
func (s *JSONSchemaProps) GetExample() string {
	ret := ""
	if s.Example != nil && len(s.Example.Raw) > 0 {
		ret = string(s.Example.Raw)
		if ret[0] == '{' {
			// string(s.Example.Raw) might look like (when the schema is generated through go src):
			// {"allow":["consumer1"],"consumers":[{"credential":"admin:123456","name":"consumer1"}]}
			var obj interface{}
			err := json.Unmarshal(s.Example.Raw, &obj)
			if err != nil {
				return ""
			}
			b, err := utils.MarshalYamlWithIndent(obj, 2)
			if err != nil {
				return ""
			}
			ret = string(b)
		}
	}
	return ret
}

// GetPropertiesOrderMap converts the schema Properties map to
// an ordered map (dictionary order) and returns it
func (s *JSONSchemaProps) GetPropertiesOrderMap() *orderedmap.OrderedMap {
	m := orderedmap.New()
	for name, prop := range s.Properties {
		m.Set(name, prop)
	}
	m.SortKeys(sort.Strings)
	return m
}

// HandleFieldAnnotations parses the comment (annotations look like `// @<KEY> [LANGUAGE] <VALUE>`)
// and sets the schema properties
func (s *JSONSchemaProps) HandleFieldAnnotations(comment string) {
	as := GetAnnotations(comment)
	for _, a := range as {
		switch a.Type {
		case ATitle:
			if s.Title == "" {
				s.Title = a.Text
			}
			s.XTitleI18n[a.I18nType] = a.Text
		case ADescription:
			if s.Description == "" {
				s.Description = a.Text
			}
			s.XDescriptionI18n[a.I18nType] = a.Text
		case AScope:
			s.Scope = Scope(a.Text)
		case AExample:
			s.Example = &JSON{Raw: []byte(a.Text)}
		}
	}
}

// HandleFieldTags parses the struct field tags and sets the schema properties
// TODO: Add more tags (now supported yaml, minimum, maximum, ...)
func (s *JSONSchemaProps) HandleFieldTags(tags string, parent *JSONSchemaProps, fieldName string) string {
	if tags == "" {
		return fieldName
	}
	st, err := structtag.Parse(tags)
	if err != nil {
		return fieldName
	}

	newName := fieldName
	for _, tag := range st.Tags() {
		switch tag.Key {
		case "yaml":
			newName = tag.Name
			if s.Title == "" {
				s.Title = newName
				s.XTitleI18n[I18nDefault] = newName
			}
		case "required":
			required, _ := strconv.ParseBool(tag.Name)
			if !required {
				continue
			}
			parent.Required = append(parent.Required, newName)
		case "minimum":
			min, err := strconv.ParseFloat(tag.Name, 64)
			if err != nil {
				continue
			}
			s.Minimum = &min
		case "maximum":
			max, err := strconv.ParseFloat(tag.Name, 64)
			if err != nil {
				continue
			}
			s.Maximum = &max
		case "minLength":
			minL, err := strconv.ParseInt(tag.Name, 10, 64)
			if err != nil {
				continue
			}
			s.MinLength = &minL
		case "maxLength":
			maxL, err := strconv.ParseInt(tag.Name, 10, 64)
			if err != nil {
				continue
			}
			s.MaxLength = &maxL
		case "minItems":
			minI, err := strconv.ParseInt(tag.Name, 10, 64)
			if err != nil {
				continue
			}
			s.MinItems = &minI
		case "maxItems":
			maxI, err := strconv.ParseInt(tag.Name, 10, 64)
			if err != nil {
				continue
			}
			s.MaxItems = &maxI
		case "pattern":
			s.Pattern = tag.Name
		}
	}

	return newName
}

// JoinRequirementsBy joins the requirements by the given i18n type. Return value looks like:
// required, minLength 10, regular expression "^.*$"
func (s *JSONSchemaProps) JoinRequirementsBy(i18n I18nType, required bool) string {
	reqs := s.getRequirements(required)
	switch i18n {
	case I18nZH_CN:
		return strings.Join(reqs[I18nZH_CN], "，")
	case I18nEN_US:
		fallthrough
	default:
		return strings.Join(reqs[I18nDefault], ", ")
	}
}

func (s *JSONSchemaProps) getRequirements(required bool) map[I18nType][]string {
	reqs := make(map[I18nType][]string)

	for i18n, str := range s.GetRequired(required) {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMinimum() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMaximum() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMinLength() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMaxLength() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMinItems() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetMaxItems() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	for i18n, str := range s.GetPattern() {
		reqs[i18n] = append(reqs[i18n], str)
	}

	return reqs
}

func (s *JSONSchemaProps) GetMinimum() map[I18nType]string {
	if s.Minimum == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最小值 %f", *s.Minimum),
		I18nEN_US: fmt.Sprintf("minimum %f", *s.Minimum),
	}
}

func (s *JSONSchemaProps) GetMaximum() map[I18nType]string {
	if s.Maximum == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最大值 %f", *s.Maximum),
		I18nEN_US: fmt.Sprintf("maximum %f", *s.Maximum),
	}
}

func (s *JSONSchemaProps) GetMinLength() map[I18nType]string {
	if s.MinLength == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最小长度 %d", *s.MinLength),
		I18nEN_US: fmt.Sprintf("minLength %d", *s.MinLength),
	}
}

func (s *JSONSchemaProps) GetMaxLength() map[I18nType]string {
	if s.MaxLength == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最大长度 %d", *s.MaxLength),
		I18nEN_US: fmt.Sprintf("maxLength %d", *s.MaxLength),
	}
}

func (s *JSONSchemaProps) GetPattern() map[I18nType]string {
	if s.Pattern == "" {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("正则表达式 %q", s.Pattern),
		I18nEN_US: fmt.Sprintf("regular expression %q", s.Pattern),
	}
}

func (s *JSONSchemaProps) GetMinItems() map[I18nType]string {
	if s.MinItems == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最小 item 个数 %d", *s.MinItems),
		I18nEN_US: fmt.Sprintf("minItems %d", *s.MinItems),
	}
}

func (s *JSONSchemaProps) GetMaxItems() map[I18nType]string {
	if s.MaxItems == nil {
		return nil
	}

	return map[I18nType]string{
		I18nZH_CN: fmt.Sprintf("最大 item 个数 %d", *s.MaxItems),
		I18nEN_US: fmt.Sprintf("maxItems %d", *s.MaxItems),
	}
}

func (s *JSONSchemaProps) GetRequired(req bool) map[I18nType]string {
	if req {
		return map[I18nType]string{
			I18nZH_CN: "必填",
			I18nEN_US: "required",
		}
	}

	return map[I18nType]string{
		I18nZH_CN: "选填",
		I18nEN_US: "optional",
	}
}

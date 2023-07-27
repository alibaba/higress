package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/structtag"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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
)

// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
type JSON struct {
	Raw []byte `json:"-" yaml:"-"`
}

func (s JSON) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		var obj interface{}
		err := json.Unmarshal(s.Raw, &obj)
		if err != nil {
			return []byte("null"), err
		}
		return json.Marshal(obj)
	}
	return []byte("null"), nil
}

func (s JSON) MarshalYAML() (interface{}, error) {
	if len(s.Raw) > 0 {
		var obj interface{}
		err := yaml.Unmarshal(s.Raw, &obj)
		if err != nil {
			return "null", err
		}
		return obj, nil
	}
	return "null", nil
}

// JSONSchemaPropsOrArray represents a value that can either be a JSONSchemaProps
// or an array of JSONSchemaProps. Mainly here for serialization purposes.
type JSONSchemaPropsOrArray struct {
	Schema      *JSONSchemaProps
	JSONSchemas []JSONSchemaProps
}

func (s JSONSchemaPropsOrArray) MarshalJSON() ([]byte, error) {
	if len(s.JSONSchemas) > 0 {
		return json.Marshal(s.JSONSchemas)
	}
	return json.Marshal(s.Schema)
}

func (s JSONSchemaPropsOrArray) MarshalYAML() (interface{}, error) {
	if len(s.JSONSchemas) > 0 {
		return s.JSONSchemas, nil
	}
	return s.Schema, nil
}

// JSONSchemaPropsOrBool represents JSONSchemaProps or a boolean value.
// Defaults to true for the boolean property.
type JSONSchemaPropsOrBool struct {
	Allows bool
	Schema *JSONSchemaProps
}

func (s JSONSchemaPropsOrBool) MarshalJSON() ([]byte, error) {
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}

	if s.Schema == nil && !s.Allows {
		return []byte("false"), nil
	}
	return []byte("true"), nil
}

func (s JSONSchemaPropsOrBool) MarshalYAML() (interface{}, error) {
	if s.Schema != nil {
		return yaml.Marshal(s.Schema)
	}

	if s.Schema == nil && !s.Allows {
		return false, nil
	}
	return true, nil
}

// JSONSchemaDependencies represent a dependencies property.
type JSONSchemaDependencies map[string]JSONSchemaPropsOrStringArray

// JSONSchemaPropsOrStringArray represents a JSONSchemaProps or a string array.
type JSONSchemaPropsOrStringArray struct {
	Schema   *JSONSchemaProps
	Property []string
}

func (s JSONSchemaPropsOrStringArray) MarshalJSON() ([]byte, error) {
	if len(s.Property) > 0 {
		return json.Marshal(s.Property)
	}
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}
	return []byte("null"), nil
}

func (s JSONSchemaPropsOrStringArray) MarshalYAML() (interface{}, error) {
	if len(s.Property) > 0 {
		return s.Property, nil
	}
	if s.Schema != nil {
		return s.Schema, nil
	}
	return "null", nil
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
		Scope:            ScopeInstance,
		XTitleI18n:       make(map[I18nType]string),
		XDescriptionI18n: make(map[I18nType]string),
	}
}

func (s *JSONSchemaProps) HandleFieldAnnotations(anns []Annotation) {
	for _, a := range anns {
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

// TODO: Add more properties
// now supported are:
// maximum, minimum
// maxLength, minLength
// pattern
// maxItems, minItems
// required
func (s *JSONSchemaProps) HandleFieldTags(tags *structtag.Tags, parent *JSONSchemaProps, fieldName string) string {
	newName := fieldName
	for _, tag := range tags.Tags() {
		switch tag.Key {
		case "yaml":
			newName = tag.Name
			if s.Title == "" {
				s.Title = newName
				s.XTitleI18n[I18nDefault] = newName
			}
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
		case "required":
			required, err := strconv.ParseBool(tag.Name)
			if err != nil {
				continue
			}
			if required {
				parent.Required = append(parent.Required, newName)
			}
		}
	}

	return newName
}

func (s *JSONSchemaProps) HandleRequirements(required bool) map[I18nType][]string {
	reqs := make(map[I18nType][]string, 0)
	reqs[I18nZH_CN] = make([]string, 0)
	reqs[I18nEN_US] = make([]string, 0)

	m := s.GetRequired(required)
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMinimum()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMaximum()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMinLength()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMaxLength()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMinItems()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetMaxItems()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	m = s.GetPattern()
	for i18n, str := range m {
		reqs[i18n] = append(reqs[i18n], str)
	}

	return reqs
}

func RequirementsJoinByI18n(reqs map[I18nType][]string, i18n I18nType) string {
	switch i18n {
	case I18nZH_CN:
		return strings.Join(reqs[i18n], "，")
	case I18nEN_US:
		return strings.Join(reqs[i18n], ", ")
	default:
		return strings.Join(reqs[i18n], "，")
	}
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
		I18nZH_CN: fmt.Sprintf("正则表达式 \"%s\"", s.Pattern),
		I18nEN_US: fmt.Sprintf("regular expression \"%s\"", s.Pattern),
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

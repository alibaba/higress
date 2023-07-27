package plugin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
)

// WasmPluginMeta is used to describe WASM plugin metadata,
// See https://higress.io/en-us/docs/user/wasm-image-spec/
type WasmPluginMeta struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Info       WasmPluginInfo `json:"info" yaml:"info"`
	Spec       WasmPluginSpec `json:"spec" yaml:"spec"`
}

func NewWasmPluginMeta(path, structName string) *WasmPluginMeta {
	fset := token.NewFileSet()
	pkgs := make(map[string]*ast.Package)
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			pkgs, err = parser.ParseDir(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to walk path: %s", path))
	}

	structs := make(map[string]*structType)
	for _, p := range pkgs {
		ss := collectStructs(p)
		for k, v := range ss {
			structs[k] = v
		}
	}

	meta := &WasmPluginMeta{
		APIVersion: "1.0.0",
		Info: WasmPluginInfo{
			Category:         CategoryCustom,
			Name:             "unnamed",
			XTitleI18n:       make(map[I18nType]string),
			XDescriptionI18n: make(map[I18nType]string),
			Version:          "0.1.0",
		},
		Spec: WasmPluginSpec{},
	}
	if model, ok := structs[structName]; ok {
		meta.genMetaFromConfigModel(structs, model)

	} else {
		panic(fmt.Sprintf("failed to find struct named %s", structName))
	}

	return meta
}

func (meta *WasmPluginMeta) genMetaFromConfigModel(structs map[string]*structType, model *structType) {
	schema := genSchemaFromType(ast.NewIdent("object"), true, structs, model)
	meta.Spec.ConfigSchema = ConfigSchema{schema}

	// fill meta.Info, meta.Spec and Schema using annotations
	meta.handleModelAnnotations(model)
}

func (meta *WasmPluginMeta) handleModelAnnotations(model *structType) {
	for _, a := range model.Annotations {
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
)

type I18nType string

const (
	I18nZH_CN     I18nType = "zh-CN" // default
	I18nEN_US     I18nType = "en-US"
	I18nUndefined I18nType = "undefined"
	I18nUnknown   I18nType = "unknown"
	I18nDefault            = I18nZH_CN
)

func str2I18nType(typ string) I18nType {
	typ = strings.ToLower(typ)
	switch typ {
	case "zh-cn":
		return I18nDefault
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
)

type ConfigSchema struct {
	OpenAPIV3Schema *JSONSchemaProps `json:"openAPIV3Schema" yaml:"openAPIV3Schema"`
}

type structType struct {
	Name        string
	Annotations []Annotation
	Node        *ast.StructType
}

func collectStructs(node ast.Node) map[string]*structType {
	structs := make(map[string]*structType, 0)
	gtxt := ""
	typ := ""
	collect := func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				gtxt = x.Doc.Text()
			}
		case *ast.TypeSpec:
			if st, ok := x.Type.(*ast.StructType); ok {
				if !st.Struct.IsValid() {
					return true
				}

				typ = x.Name.String()
				txt := x.Doc.Text()
				if txt == "" && gtxt != "" {
					txt = gtxt
					gtxt = ""
				}
				if s, ok := structs[typ]; ok {
					s.Annotations = getAnnotations(strings.Split(txt, "\n"))
				} else {
					s := &structType{
						Name:        typ,
						Annotations: getAnnotations(strings.Split(txt, "\n")),
						Node:        x.Type.(*ast.StructType),
					}
					structs[s.Name] = s
				}
			}
		}

		return true
	}

	ast.Inspect(node, collect)
	return structs
}

func genSchemaFromType(typ ast.Expr, isStruct bool, structs map[string]*structType, stc *structType) *JSONSchemaProps {
	schema := NewJSONSchemaProps()

	if isStruct {
		schema.Type = "object"
		// 遍历成员变量，通过字段名、类型、注解和 tag 设置相应 schema 字段和属性
		handleFields(structs, schema, stc)

	} else {
		if id, ok := typ.(*ast.Ident); ok {
			ft := id.Name
			switch ft {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64":
				schema.Type = "integer"
			case "float32", "float64":
				schema.Type = "number"
			case "bool":
				schema.Type = "boolean"
			case "string":
				schema.Type = "string"
			default:
				panic("unsupported type: " + reflect.TypeOf(typ).String())
			}

		} else if at, ok := typ.(*ast.ArrayType); ok {
			schema.Type = "array"
			typeName := at.Elt.(*ast.Ident).Name
			var item *JSONSchemaProps
			if s, ok := structs[typeName]; ok {
				item = genSchemaFromType(at.Elt, true, structs, s)
			} else {
				item = genSchemaFromType(at.Elt, false, structs, nil)
			}
			schema.Items = &JSONSchemaPropsOrArray{
				Schema: item,
			}
		}
	}

	return schema
}

func handleFields(structs map[string]*structType, parent *JSONSchemaProps, stc *structType) {
	parent.Properties = make(map[string]JSONSchemaProps)

	for _, field := range stc.Node.Fields.List {
		// 1. 获取字段名称，设置为默认 title
		var fieldName string
		if field.Names == nil {
			continue
		}
		for _, name := range field.Names {
			if name.String() != "" {
				fieldName = name.String()
				break
			}
		}

		schema := &JSONSchemaProps{
			XTitleI18n:       make(map[I18nType]string),
			XDescriptionI18n: make(map[I18nType]string),
		}
		if child, ok := structs[fieldName]; ok {
			schema = genSchemaFromType(field.Type, true, structs, child)
		} else {
			schema = genSchemaFromType(field.Type, false, structs, nil)
		}

		// 3. 解析 Annotations 得到 title 和 description
		if field.Doc != nil {
			anns := getAnnotations(strings.Split(strings.TrimSpace(field.Doc.Text()), "\n"))
			schema.HandleFieldAnnotations(anns)
		}

		// 4. 解析 Tags 设置相关属性字段
		var newName string
		if field.Tag != nil {
			tags, err := structtag.Parse(strings.Trim(field.Tag.Value, "`"))
			if err != nil {
				continue
			}
			newName = schema.HandleFieldTags(tags, parent, fieldName)
		}

		parent.Properties[newName] = *schema
	}
}

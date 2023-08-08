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
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
	"github.com/pkg/errors"
)

// WasmPluginMeta is used to describe WASM plugin metadata,
// See https://higress.io/en-us/docs/user/wasm-image-spec/
type WasmPluginMeta struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Info       WasmPluginInfo `json:"info" yaml:"info"`
	Spec       WasmPluginSpec `json:"spec" yaml:"spec"`
}

func NewWasmPluginMeta(path, structName string) (*WasmPluginMeta, error) {
	fset := token.NewFileSet()
	pkgs := make(map[string]*ast.Package)
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			pds, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			for k, v := range pds {
				pkgs[k] = v
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk path: %q", path)
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
		Spec: WasmPluginSpec{
			Phase:    PhaseUnspecified,
			Priority: 0,
		},
	}

	if model, ok := structs[structName]; ok {
		meta.genMetaFromConfigModel(structs, model)
	} else {
		return nil, errors.Errorf("failed to find struct named: %q", structName)
	}

	return meta, nil
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

const (
	IconAuth        = "https://img.alicdn.com/imgextra/i4/O1CN01BPFGlT1pGZ2VDLgaH_!!6000000005333-2-tps-42-42.png"
	IconSecurity    = "https://img.alicdn.com/imgextra/i1/O1CN01jKT9vC1O059vNaq5u_!!6000000001642-2-tps-42-42.png"
	IconProtocol    = "https://img.alicdn.com/imgextra/i2/O1CN01xIywow1mVGuRUjbhe_!!6000000004959-2-tps-42-42.png"
	IconFlowControl = "https://img.alicdn.com/imgextra/i3/O1CN01bAFa9k1t1gdQcVTH0_!!6000000005842-2-tps-42-42.png"
	IconFlowMonitor = "https://img.alicdn.com/imgextra/i4/O1CN01aet3s61MoLOEEhRIo_!!6000000001481-2-tps-42-42.png"
	IconCustom      = "https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png"
)

// TODO: Change the map associated with I18nType to an ordered map, e.g., using wrapped github.com/iancoleman/orderedmap. The aim is to keep the generated files stable.

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
					s.Annotations = GetAnnotations(strings.Split(txt, "\n"))
				} else {
					s := &structType{
						Name:        typ,
						Annotations: GetAnnotations(strings.Split(txt, "\n")),
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

const unknownIsStruct = false

// TODO: More types need to be supported, e.g., Map, Pointer ...
func genSchemaFromType(typ ast.Expr, isStruct bool, structs map[string]*structType, stc *structType) *JSONSchemaProps {
	schema := NewJSONSchemaProps()

	if isStruct { // explicitly declare that it is a struct
		schema.Type = "object"
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
				if s, ok := structs[ft]; ok { // implicitly declare that it is a struct
					schema.Type = "object"
					handleFields(structs, schema, s)
				} else {
					panic("unsupported type: " + ft)
				}
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

// iterate over the fields, setting the corresponding
// schema fields and properties by field name, type, annotations, tags, etc.
func handleFields(structs map[string]*structType, parent *JSONSchemaProps, stc *structType) {
	parent.Properties = make(map[string]JSONSchemaProps)

	for _, field := range stc.Node.Fields.List {
		// 1. get filed name as the default key name for the schema property
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

		// 2. get the schema of the field
		schema := genSchemaFromType(field.Type, unknownIsStruct, structs, nil)

		// 3. parse the annotations of the field
		if field.Doc != nil {
			anns := GetAnnotations(strings.Split(strings.TrimSpace(field.Doc.Text()), "\n"))
			schema.HandleFieldAnnotations(anns)
		}

		// 4. parse the tags of the field
		newName := fieldName
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

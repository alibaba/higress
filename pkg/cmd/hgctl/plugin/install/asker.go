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

package install

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/iancoleman/orderedmap"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	askInterrupted   = "X Interrupted."
	invalidSyntax    = "X Invalid syntax."
	failedToValidate = "X Failed to validate: not satisfied with schema."

	addConfSuccessful = "√ Successful to add configuration."

	iconIdent = strings.Repeat(" ", 2)
)

type Asker interface {
	Ask() error
}

type WasmPluginSpecConfAsker struct {
	resp *WasmPluginSpecConf

	ingask *IngressAsker
	domask *DomainAsker
	glcask *GlobalConfAsker

	printer *Printer
}

func NewWasmPluginSpecConfAsker(ingask *IngressAsker, domask *DomainAsker, glcask *GlobalConfAsker, printer *Printer) *WasmPluginSpecConfAsker {
	return &WasmPluginSpecConfAsker{
		ingask:  ingask,
		domask:  domask,
		glcask:  glcask,
		printer: printer,
	}
}

func (p *WasmPluginSpecConfAsker) Ask() error {
	var (
		wpc = NewPluginSpecConf()

		globalConf  map[string]interface{}
		ingressRule *IngressMatchRule
		domainRule  *DomainMatchRule

		scopea    = newScopeAsker(p.printer)
		continuea = newContinueAsker(p.printer)
		rewritea  = newRewriteAsker(p.printer)
		rulea     = newRuleAsker(p.printer)
	)

	for {
		err := scopea.Ask()
		if err != nil {
			return err
		}
		scope := scopea.resp

		switch scope {
		case types.ScopeInstance:
			err = rulea.Ask()
			if err != nil {
				return err
			}
			rule := rulea.resp

			switch rule {
			case RuleIngress:
				if ingressRule != nil {
					p.printer.Yesf("\n%s\n", ingressRule)
					err = rewritea.Ask()
					if err != nil {
						return err
					}
					if !rewritea.resp {
						continue
					}
				}

				p.ingask.scope = scope
				err = p.ingask.Ask()
				if err != nil {
					return err
				}
				ingressRule = p.ingask.resp

			case RuleDomain:
				if domainRule != nil {
					p.printer.Yesf("\n%s\n", domainRule)
					err = rewritea.Ask()
					if err != nil {
						return err
					}
					if !rewritea.resp {
						continue
					}
				}

				p.domask.scope = scope
				err = p.domask.Ask()
				if err != nil {
					return err
				}
				domainRule = p.domask.resp
			}

		case types.ScopeGlobal:
			if globalConf != nil {
				b, _ := utils.MarshalYamlWithIndent(globalConf, 2)
				p.printer.Yesf("\n%s\n", string(b))
				err = rewritea.Ask()
				if err != nil {
					return err
				}
				if !rewritea.resp {
					continue
				}
			}

			p.glcask.scope = scope
			err = p.glcask.Ask()
			if err != nil {
				return err
			}
			globalConf = p.glcask.resp
		}

		err = continuea.Ask()
		if err != nil {
			return err
		}
		if !continuea.resp {
			break
		}
	}

	if globalConf != nil {
		wpc.DefaultConfig = globalConf
	}
	if ingressRule != nil {
		wpc.MatchRules = append(wpc.MatchRules, ingressRule)
	}
	if domainRule != nil {
		wpc.MatchRules = append(wpc.MatchRules, domainRule)
	}

	p.printer.Yesln("The complete configuration is as follows:")
	p.printer.Yesf("\n%s\n", wpc)
	p.resp = wpc
	return nil
}

type IngressAsker struct {
	resp *IngressMatchRule

	structName string
	schema     *types.JSONSchemaProps
	scope      types.Scope

	vld *jsonschema.Schema // for validation

	printer *Printer
}

func NewIngressAsker(structName string, schema *types.JSONSchemaProps, vld *jsonschema.Schema, printer *Printer) *IngressAsker {
	return &IngressAsker{
		structName: structName,
		schema:     schema,
		vld:        vld,
		printer:    printer,
	}
}

func (i *IngressAsker) Ask() error {
	continuea := newContinueAsker(i.printer)
	ings := make([]string, 0)
	for {
		var ing string
		err := survey.AskOne(&survey.Input{
			Message: "Enter the matched ingress:",
			Help:    "Matching ingress resource object, the matching format is: namespace/ingress name",
		}, &ing)
		if err != nil {
			return err
		}

		ing = strings.TrimSpace(ing)
		if ing != "" {
			ings = append(ings, ing)
		}

		err = continuea.Ask()
		if err != nil {
			return err
		}
		if !continuea.resp {
			break
		}
	}

	i.printer.Yesln(iconIdent + "Ingress:")
	as, err := recursivePrompt(i.structName, i.schema, i.scope, i.printer)
	if err != nil {
		return err
	}
	if ok, ve := validate(i.vld, as); !ok {
		i.printer.Noln(failedToValidate)
		i.printer.Noln(ve)
		return nil
	}

	i.resp = &IngressMatchRule{
		Ingress: ings,
		Config:  as,
	}
	i.printer.Yesln(addConfSuccessful)

	return nil
}

type DomainAsker struct {
	resp *DomainMatchRule

	structName string
	schema     *types.JSONSchemaProps
	scope      types.Scope

	vld *jsonschema.Schema // for validation

	printer *Printer
}

func NewDomainAsker(structName string, schema *types.JSONSchemaProps, vld *jsonschema.Schema, printer *Printer) *DomainAsker {
	return &DomainAsker{
		structName: structName,
		schema:     schema,
		vld:        vld,
		printer:    printer,
	}
}

func (d *DomainAsker) Ask() error {
	continuea := newContinueAsker(d.printer)
	doms := make([]string, 0)
	for {
		var dom string
		err := survey.AskOne(&survey.Input{
			Message: "Enter the matched domain:",
			Help:    "match domain name, support generic domain name",
		}, &dom)
		if err != nil {
			return err
		}

		dom = strings.TrimSpace(dom)
		if dom != "" {
			doms = append(doms, dom)
		}

		err = continuea.Ask()
		if err != nil {
			return err
		}
		if !continuea.resp {
			break
		}
	}

	d.printer.Yesln(iconIdent + "Domain:")
	as, err := recursivePrompt(d.structName, d.schema, d.scope, d.printer)
	if err != nil {
		return err
	}
	if ok, ve := validate(d.vld, as); !ok {
		d.printer.Noln(failedToValidate)
		d.printer.Noln(ve)
		return nil
	}

	d.resp = &DomainMatchRule{
		Domain: doms,
		Config: as,
	}
	d.printer.Yesln(addConfSuccessful)

	return nil
}

type GlobalConfAsker struct {
	resp map[string]interface{}

	structName string
	schema     *types.JSONSchemaProps
	scope      types.Scope

	vld *jsonschema.Schema // for validation

	printer *Printer
}

func NewGlobalConfAsker(structName string, schema *types.JSONSchemaProps, vld *jsonschema.Schema, printer *Printer) *GlobalConfAsker {
	return &GlobalConfAsker{
		structName: structName,
		schema:     schema,
		vld:        vld,
		printer:    printer,
	}
}

func (g *GlobalConfAsker) Ask() error {
	g.printer.Yesln(iconIdent + "Global:")
	as, err := recursivePrompt(g.structName, g.schema, g.scope, g.printer)
	if err != nil {
		return err
	}
	if ok, ve := validate(g.vld, as); !ok {
		g.printer.Noln(failedToValidate)
		g.printer.Noln(ve)
		return nil
	}

	g.resp = as.(map[string]interface{})
	g.printer.Yesln(addConfSuccessful)

	return nil
}

type continueAsker struct {
	resp bool

	printer *Printer
}

func newContinueAsker(printer *Printer) *continueAsker {
	return &continueAsker{printer: printer}
}

func (c *continueAsker) Ask() error {
	resp := true
	err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("%scontinue?", c.printer.Ident()),
		Default: true,
	}, &resp)
	if err != nil {
		return err
	}

	c.resp = resp
	return nil
}

type rewriteAsker struct {
	resp bool

	printer *Printer
}

func newRewriteAsker(printer *Printer) *rewriteAsker {
	return &rewriteAsker{printer: printer}
}

func (r *rewriteAsker) Ask() error {
	resp := false
	err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("%sThe configuration already exists as shown above. Do you want to rewrite it?", r.printer.Ident()),
		Default: false,
	}, &resp)
	if err != nil {
		return err
	}

	r.resp = resp
	return nil
}

type scopeAsker struct {
	resp types.Scope

	printer *Printer
}

func newScopeAsker(printer *Printer) *scopeAsker {
	return &scopeAsker{printer: printer}
}

func (s *scopeAsker) Ask() error {
	var resp string
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("%sChoose a configuration effective scope:", s.printer.Ident()),
		Options: []string{
			string(types.ScopeInstance),
			string(types.ScopeGlobal),
		},
		Default: string(types.ScopeInstance),
	}, &resp)
	if err != nil {
		return err
	}

	s.resp = types.Scope(resp)
	return nil
}

type ruleAsker struct {
	resp Rule

	printer *Printer
}

func newRuleAsker(printer *Printer) *ruleAsker {
	return &ruleAsker{printer: printer}
}

func (r *ruleAsker) Ask() error {
	var resp string
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("%sChoose Ingress or Domain:", r.printer.Ident()),
		Options: []string{
			string(RuleIngress),
			string(RuleDomain),
		},
		Default: string(RuleIngress),
	}, &resp)
	if err != nil {
		return err
	}

	r.resp = Rule(resp)
	return nil
}

type WasmPluginSpecConf struct {
	DefaultConfig map[string]interface{} `yaml:"defaultConfig,omitempty"`
	MatchRules    []MatchRule            `yaml:"matchRules,omitempty"`
}

func NewPluginSpecConf() *WasmPluginSpecConf {
	return &WasmPluginSpecConf{
		MatchRules: make([]MatchRule, 0),
	}
}

func (p *WasmPluginSpecConf) String() string {
	b, _ := utils.MarshalYamlWithIndent(p, 2)
	return string(b)
}

type MatchRule interface {
	String() string
}

type IngressMatchRule struct {
	Ingress []string    `json:"ingress" yaml:"ingress" mapstructure:"ingress"`
	Config  interface{} `json:"config" yaml:"config" mapstructure:"config"`
}

func (i IngressMatchRule) String() string {
	b, _ := utils.MarshalYamlWithIndent(i, 2)
	return string(b)
}

func decodeIngressMatchRule(obj map[string]interface{}) (*IngressMatchRule, error) {
	var ing IngressMatchRule
	if err := mapstructure.Decode(obj, &ing); err != nil {
		return nil, err
	}

	return &ing, nil
}

type DomainMatchRule struct {
	Domain []string    `json:"domain" yaml:"domain" mapstructure:"domain"`
	Config interface{} `json:"config" yaml:"config" mapstructure:"config"`
}

func (d DomainMatchRule) String() string {
	b, _ := utils.MarshalYamlWithIndent(d, 2)
	return string(b)
}

func decodeDomainMatchRule(obj map[string]interface{}) (*DomainMatchRule, error) {
	var dom DomainMatchRule
	if err := mapstructure.Decode(obj, &dom); err != nil {
		return nil, err
	}

	return &dom, nil
}

type Rule string

const (
	RuleIngress Rule = "Ingress"
	RuleDomain  Rule = "Domain"
)

func recursivePrompt(structName string, schema *types.JSONSchemaProps, selScope types.Scope, printer *Printer) (interface{}, error) {
	printer.IncIdentRepeat()
	defer printer.DecIndentRepeat()
	return doPrompt(structName, nil, schema, types.ScopeAll, selScope, printer)
}

func doPrompt(fieldName string, parent, schema *types.JSONSchemaProps, oriScope, selScope types.Scope, printer *Printer) (interface{}, error) {
	if schema.Title == "" {
		schema.Title = fieldName
	}
	if schema.Description == "" {
		schema.Description = fieldName
	}
	required := true
	if parent != nil {
		required = isRequired(fieldName, parent.Required)
	}
	msg, help := fieldTips(fieldName, parent, schema, required, printer)

	switch schema.Type {
	case "object":
		printer.Println(iconIdent + msg)
		obj := make(map[string]interface{})
		odProps := orderedmap.New()
		for name, prop := range schema.Properties {
			if name == "" {
				continue
			}
			odProps.Set(name, prop)
		}
		odProps.SortKeys(sort.Strings)
		for _, name := range odProps.Keys() {
			propI, _ := odProps.Get(name)
			prop := propI.(types.JSONSchemaProps)

			if parent == nil { // keep topmost scope
				if prop.Scope == types.ScopeGlobal {
					oriScope = types.ScopeGlobal
				} else if prop.Scope == types.ScopeInstance || prop.Scope == "" {
					oriScope = types.ScopeInstance
				}
			}

			if !matchesScope(oriScope, selScope, prop.Scope) {
				continue
			}

			printer.IncIdentRepeat()
			v, err := doPrompt(name, schema, &prop, oriScope, selScope, printer)
			printer.DecIndentRepeat()
			if err != nil {
				return nil, err
			}
			if v != nil {
				obj[name] = v
			}
		}

		return obj, nil

	case "array":
		printer.Println(iconIdent + msg)
		continuea := newContinueAsker(printer)
		arr := make([]interface{}, 0)
		for {
			printer.IncIdentRepeat()
			v, err := doPrompt("item", schema, schema.Items.Schema, oriScope, selScope, printer)
			if err != nil {
				printer.DecIndentRepeat()
				return nil, err
			}
			if v != nil {
				arr = append(arr, v)
			}

			err = continuea.Ask()
			printer.DecIndentRepeat()
			if err != nil {
				return nil, err
			}

			if !continuea.resp {
				break
			}
		}

		return arr, nil

	case "integer", "number", "boolean", "string":
		for {
			var inp string
			if err := survey.AskOne(&survey.Input{
				Message: msg,
				Help:    help,
			}, &inp); err != nil {
				return nil, err
			}
			if inp == "" && !required {
				return nil, nil
			}

			switch schema.Type {
			case "integer":
				v, err := strconv.ParseInt(inp, 10, 64)
				if err != nil {
					if errors.Is(err, strconv.ErrSyntax) {
						printer.Nof("%s %q type is invalid.\n", invalidSyntax, inp)
						continue
					}
					return nil, err
				}
				return v, nil
			case "number":
				v, err := strconv.ParseFloat(inp, 64)
				if err != nil {
					if errors.Is(err, strconv.ErrSyntax) {
						printer.Nof("%s %q type is invalid.\n", invalidSyntax, inp)
						continue
					}
					return nil, err
				}
				return v, nil
			case "boolean":
				v, err := strconv.ParseBool(inp)
				if err != nil {
					if errors.Is(err, strconv.ErrSyntax) {
						printer.Nof("%s %q type is invalid.\n", invalidSyntax, inp)
						continue
					}
					return nil, err
				}
				return v, nil
			case "string":
				return inp, nil
			default:
				return inp, nil
			}
		}

	default:
		return nil, fmt.Errorf("unsupported type: %s", schema.Type)
	}
}

func matchesScope(oriScope, selScope, scope types.Scope) bool {
	return (oriScope == selScope) ||
		(selScope == types.ScopeInstance && (scope == selScope || scope == "" || scope == types.ScopeAll)) ||
		(selScope == types.ScopeGlobal && (scope == selScope || scope == types.ScopeAll))
}

func fieldTips(fieldName string, parent, schema *types.JSONSchemaProps, required bool, printer *Printer) (string, string) {
	var msg, help string
	if fieldName == "item" {
		msg = fmt.Sprintf("%s%s(%s)", printer.Ident(), fieldName, schema.Type)
		help = fmt.Sprintf("%s%s: %s", printer.Ident(), parent.Title, parent.Description)
	} else {
		reqs := schema.HandleRequirements(required)
		req := types.RequirementsJoinByI18n(reqs, types.I18nEN_US)
		msg = fmt.Sprintf("%s%s(%s, %s)", printer.Ident(), fieldName, schema.Type, req)
		help = fmt.Sprintf("%s%s: %s", printer.Ident(), schema.Title, schema.Description)
	}

	return msg, help
}

func isRequired(name string, required []string) bool {
	req := false
	for _, n := range required {
		if name == n {
			req = true
			break
		}
	}
	return req
}

func validate(schema *jsonschema.Schema, v interface{}) (bool, error) {
	if err := schema.Validate(v); err != nil {
		err = convertValidationError(err.(*jsonschema.ValidationError))
		return false, err
	}
	return true, nil
}

func convertValidationError(ve *jsonschema.ValidationError) error {
	de := ve.DetailedOutput()
	if de.Valid {
		return nil
	}

	errs := make([]error, 0)
	if de.Error != "" {
		errs = append(errs, errors.New(de.Error))
	}
	errs = append(errs, doConvertValidationError(de.Errors, errs)...)
	if len(errs) == 0 {
		return nil
	}

	var ret error
	for i, err := range errs {
		if i == 0 {
			ret = fmt.Errorf("%w", err)
		} else {
			ret = fmt.Errorf("%s\n%w", ret.Error(), err)
		}
	}
	return ret
}

func doConvertValidationError(de []jsonschema.Detailed, errs []error) []error {
	for _, e := range de {
		if e.Error != "" {
			errs = append(errs, errors.New(e.Error))
		}
		if len(e.Errors) > 0 {
			errs = append(errs, doConvertValidationError(e.Errors, errs)...)
		}
	}
	return errs
}

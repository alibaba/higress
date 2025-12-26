package pkg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/tidwall/gjson"
)

const (
	conditionTypeEquals   = "equals"
	conditionTypePrefix   = "prefix"
	conditionTypeSuffix   = "suffix"
	conditionTypeContains = "contains"
	conditionTypeRegex    = "regex"
)

var (
	conditionFactories = map[string]func(gjson.Result) (Condition, error){
		conditionTypeEquals:   newEqualsCondition,
		conditionTypePrefix:   newPrefixCondition,
		conditionTypeSuffix:   newSuffixCondition,
		conditionTypeContains: newContainsCondition,
		conditionTypeRegex:    newRegexCondition,
	}
)

type ConditionSet struct {
	Conditions    []Condition    `json:"conditions,omitempty"`
	RelatedStages map[Stage]bool `json:"-"`
}

func (s *ConditionSet) FromJson(json gjson.Result) error {
	relatedStages := map[Stage]bool{}
	s.Conditions = nil
	if conditionsJson := json.Get("conditions"); conditionsJson.Exists() && conditionsJson.IsArray() {
		for _, item := range conditionsJson.Array() {
			if condition, err := CreateCondition(item); err != nil {
				return fmt.Errorf("failed to create condition from json: %v\n  %v", err, item)
			} else {
				s.Conditions = append(s.Conditions, condition)
				for _, ref := range condition.GetRefs() {
					relatedStages[ref.GetStage()] = true
				}
			}
		}
	}
	s.RelatedStages = relatedStages

	return nil
}

func (s *ConditionSet) Matches(editorContext EditorContext) bool {
	if len(s.Conditions) == 0 {
		return true
	}
	for _, condition := range s.Conditions {
		if !condition.Evaluate(editorContext) {
			return false
		}
	}
	return true
}

type Condition interface {
	GetType() string
	GetRefs() []*Ref
	Evaluate(ctx EditorContext) bool
}

func CreateCondition(json gjson.Result) (Condition, error) {
	t := json.Get("type").String()
	if t == "" {
		return nil, errors.New("condition type is required")
	}
	if constructor, ok := conditionFactories[t]; !ok || constructor == nil {
		return nil, errors.New("unknown condition type: " + t)
	} else if condition, err := constructor(json); err != nil {
		return nil, fmt.Errorf("failed to create condition with type %s: %v", t, err)
	} else {
		for _, ref := range condition.GetRefs() {
			if ref.GetStage() >= StageResponseHeaders {
				return nil, fmt.Errorf("condition only supports request refs")
			}
		}
		return condition, nil
	}
}

// equalsCondition
func newEqualsCondition(json gjson.Result) (Condition, error) {
	value1 := json.Get("value1")
	if value1.Type != gjson.JSON {
		return nil, errors.New("equalsCondition: value1 field type must be JSON object")
	}
	value1Ref, err := NewRef(value1)
	if err != nil {
		return nil, errors.New("equalsCondition: failed to create value1 ref: " + err.Error())
	}
	value2 := json.Get("value2").String()
	return &equalsCondition{
		value1Ref: value1Ref,
		value2:    value2,
	}, nil
}

type equalsCondition struct {
	value1Ref *Ref
	value2    string
}

func (c *equalsCondition) GetType() string {
	return conditionTypeEquals
}

func (c *equalsCondition) GetRefs() []*Ref {
	return []*Ref{c.value1Ref}
}

func (c *equalsCondition) Evaluate(ctx EditorContext) bool {
	log.Debugf("Evaluating equals condition: value1Ref=%v, value2=%s", c.value1Ref, c.value2)
	ref1Values := ctx.GetRefValues(c.value1Ref)
	if len(ref1Values) == 0 {
		log.Debugf("No values found for ref1: %v", c.value1Ref)
		return false
	}
	for _, value1 := range ref1Values {
		if value1 == c.value2 {
			log.Debugf("Condition matched: %s == %s", value1, c.value2)
			return true
		}
	}
	log.Debugf("No matches found for condition: value1Ref=%v, value2=%s", c.value1Ref, c.value2)
	return false
}

// prefixCondition
func newPrefixCondition(json gjson.Result) (Condition, error) {
	value := json.Get("value")
	if value.Type != gjson.JSON {
		return nil, errors.New("prefixCondition: value field type must be JSON object")
	}
	valueRef, err := NewRef(value)
	if err != nil {
		return nil, errors.New("prefixCondition: failed to create value ref: " + err.Error())
	}
	prefix := json.Get("prefix").String()
	return &prefixCondition{
		valueRef: valueRef,
		prefix:   prefix,
	}, nil
}

type prefixCondition struct {
	valueRef *Ref
	prefix   string
}

func (c *prefixCondition) GetType() string {
	return conditionTypePrefix
}

func (c *prefixCondition) GetRefs() []*Ref {
	return []*Ref{c.valueRef}
}

func (c *prefixCondition) Evaluate(ctx EditorContext) bool {
	log.Debugf("Evaluating prefix condition: valueRef=%v, prefix=%s", c.valueRef, c.prefix)
	refValues := ctx.GetRefValues(c.valueRef)
	if len(refValues) == 0 {
		log.Debugf("No values found for ref: %v", c.valueRef)
		return false
	}
	for _, value := range refValues {
		if strings.HasPrefix(value, c.prefix) {
			log.Debugf("Condition matched: %s starts with %s", value, c.prefix)
			return true
		}
	}
	log.Debugf("No matches found for condition: valueRef=%v, prefix=%s", c.valueRef, c.prefix)
	return false
}

// suffixCondition
func newSuffixCondition(json gjson.Result) (Condition, error) {
	value := json.Get("value")
	if value.Type != gjson.JSON {
		return nil, errors.New("suffixCondition: value field type must be JSON object")
	}
	valueRef, err := NewRef(value)
	if err != nil {
		return nil, errors.New("suffixCondition: failed to create value ref: " + err.Error())
	}
	suffix := json.Get("suffix").String()
	return &suffixCondition{
		valueRef: valueRef,
		suffix:   suffix,
	}, nil
}

type suffixCondition struct {
	valueRef *Ref
	suffix   string
}

func (c *suffixCondition) GetType() string {
	return conditionTypeSuffix
}

func (c *suffixCondition) GetRefs() []*Ref {
	return []*Ref{c.valueRef}
}
func (c *suffixCondition) Evaluate(ctx EditorContext) bool {
	log.Debugf("Evaluating suffix condition: valueRef=%v, prefix=%s", c.valueRef, c.suffix)
	refValues := ctx.GetRefValues(c.valueRef)
	if len(refValues) == 0 {
		log.Debugf("No values found for ref: %v", c.valueRef)
		return false
	}
	for _, value := range refValues {
		if strings.HasSuffix(value, c.suffix) {
			log.Debugf("Condition matched: %s ends with %s", value, c.suffix)
			return true
		}
	}
	log.Debugf("No matches found for condition: valueRef=%v, prefix=%s", c.valueRef, c.suffix)
	return false
}

// containsCondition
func newContainsCondition(json gjson.Result) (Condition, error) {
	value := json.Get("value")
	if value.Type != gjson.JSON {
		return nil, errors.New("containsCondition: value field type must be JSON object")
	}
	valueRef, err := NewRef(value)
	if err != nil {
		return nil, errors.New("containsCondition: failed to create value ref: " + err.Error())
	}
	part := json.Get("part").String()
	return &containsCondition{
		valueRef: valueRef,
		part:     part,
	}, nil
}

type containsCondition struct {
	valueRef *Ref
	part     string
}

func (c *containsCondition) GetType() string {
	return conditionTypeContains
}

func (c *containsCondition) GetRefs() []*Ref {
	return []*Ref{c.valueRef}
}

func (c *containsCondition) Evaluate(ctx EditorContext) bool {
	refValues := ctx.GetRefValues(c.valueRef)
	if len(refValues) == 0 {
		return false
	}
	for _, value := range refValues {
		if strings.Contains(value, c.part) {
			return true
		}
	}
	return false
}

// regexCondition
func newRegexCondition(json gjson.Result) (Condition, error) {
	value := json.Get("value")
	if value.Type != gjson.JSON {
		return nil, errors.New("regexCondition: value field type must be JSON object")
	}
	valueRef, err := NewRef(value)
	if err != nil {
		return nil, errors.New("regexCondition: failed to create value ref: " + err.Error())
	}
	patternStr := json.Get("pattern").String()
	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, errors.New("regexCondition: failed to compile pattern: " + err.Error())
	}
	return &regexCondition{
		valueRef: valueRef,
		pattern:  pattern,
	}, nil
}

type regexCondition struct {
	valueRef *Ref
	pattern  *regexp.Regexp
}

func (c *regexCondition) GetType() string {
	return conditionTypeRegex
}

func (c *regexCondition) Evaluate(ctx EditorContext) bool {
	log.Debugf("Evaluating regex condition: valueRef=%v, pattern=%s", c.valueRef, c.pattern.String())
	refValues := ctx.GetRefValues(c.valueRef)
	if len(refValues) == 0 {
		log.Debugf("No values found for ref: %v", c.valueRef)
		return false
	}
	for _, value := range refValues {
		if c.pattern.MatchString(value) {
			log.Debugf("Condition matched: %s matches %s", value, c.pattern.String())
			return true
		}
	}
	log.Debugf("No matches found for condition: valueRef=%v, pattern=%s", c.valueRef, c.pattern.String())
	return false
}

func (c *regexCondition) GetRefs() []*Ref {
	return []*Ref{c.valueRef}
}

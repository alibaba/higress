package pkg

import (
	"testing"

	"github.com/tidwall/gjson"
)

// --- equalsCondition tests ---
func TestEqualsCondition_Match(t *testing.T) {
	json := gjson.Parse(`{"type":"equals","value1":{"type":"request_header","name":"x-test"},"value2":"abc"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-test": {"abc"}})
	if !cond.Evaluate(ctx) {
		t.Error("equalsCondition should match")
	}
}

func TestEqualsCondition_NoMatch(t *testing.T) {
	json := gjson.Parse(`{"type":"equals","value1":{"type":"request_header","name":"x-test"},"value2":"abc"}`)
	cond, _ := CreateCondition(json)
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-test": {"def"}})
	if cond.Evaluate(ctx) {
		t.Error("equalsCondition should not match")
	}
}

// --- prefixCondition tests ---
func TestPrefixCondition_Match(t *testing.T) {
	json := gjson.Parse(`{"type":"prefix","value":{"type":"request_query","name":"foo"},"prefix":"bar"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestQueries(map[string][]string{"foo": {"barbaz"}})
	if !cond.Evaluate(ctx) {
		t.Error("prefixCondition should match")
	}
}

func TestPrefixCondition_NoMatch(t *testing.T) {
	json := gjson.Parse(`{"type":"prefix","value":{"type":"request_query","name":"foo"},"prefix":"bar"}`)
	cond, _ := CreateCondition(json)
	ctx := NewEditorContext()
	ctx.SetRequestQueries(map[string][]string{"foo": {"bazbar"}})
	if cond.Evaluate(ctx) {
		t.Error("prefixCondition should not match")
	}
}

// --- suffixCondition tests ---
func TestSuffixCondition_Match(t *testing.T) {
	json := gjson.Parse(`{"type":"suffix","value":{"type":"request_header","name":"x-end"},"suffix":"xyz"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-end": {"123xyz"}})
	if !cond.Evaluate(ctx) {
		t.Error("suffixCondition should match")
	}
}

func TestSuffixCondition_NoMatch(t *testing.T) {
	json := gjson.Parse(`{"type":"suffix","value":{"type":"request_header","name":"x-end"},"suffix":"xyz"}`)
	cond, _ := CreateCondition(json)
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-end": {"xyz123"}})
	if cond.Evaluate(ctx) {
		t.Error("suffixCondition should not match")
	}
}

// --- containsCondition tests ---
func TestContainsCondition_Match(t *testing.T) {
	json := gjson.Parse(`{"type":"contains","value":{"type":"request_query","name":"foo"},"part":"baz"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestQueries(map[string][]string{"foo": {"barbaz"}})
	if !cond.Evaluate(ctx) {
		t.Error("containsCondition should match")
	}
}

func TestContainsCondition_NoMatch(t *testing.T) {
	json := gjson.Parse(`{"type":"contains","value":{"type":"request_query","name":"foo"},"part":"baz"}`)
	cond, _ := CreateCondition(json)
	ctx := NewEditorContext()
	ctx.SetRequestQueries(map[string][]string{"foo": {"bar"}})
	if cond.Evaluate(ctx) {
		t.Error("containsCondition should not match")
	}
}

// --- regexCondition tests ---
func TestRegexCondition_Match(t *testing.T) {
	json := gjson.Parse(`{"type":"regex","value":{"type":"request_header","name":"x-reg"},"pattern":"^abc.*"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-reg": {"abcdef"}})
	if !cond.Evaluate(ctx) {
		t.Error("regexCondition should match")
	}
}

func TestRegexCondition_NoMatch(t *testing.T) {
	json := gjson.Parse(`{"type":"regex","value":{"type":"request_header","name":"x-reg"},"pattern":"^abc.*"}`)
	cond, _ := CreateCondition(json)
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-reg": {"defabc"}})
	if cond.Evaluate(ctx) {
		t.Error("regexCondition should not match")
	}
}

// --- CreateCondition error cases ---
func TestCreateCondition_UnknownType(t *testing.T) {
	json := gjson.Parse(`{"type":"unknown","value1":{"type":"request_header","name":"x-test"},"value2":"abc"}`)
	_, err := CreateCondition(json)
	if err == nil {
		t.Error("CreateCondition should fail for unknown type")
	}
}

func TestCreateCondition_MissingType(t *testing.T) {
	json := gjson.Parse(`{"value1":{"type":"request_header","name":"x-test"},"value2":"abc"}`)
	_, err := CreateCondition(json)
	if err == nil {
		t.Error("CreateCondition should fail for missing type")
	}
}

func TestCreateCondition_InvalidRefType(t *testing.T) {
	json := gjson.Parse(`{"type":"equals","value1":{"type":"invalid_type","name":"x-test"},"value2":"abc"}`)
	_, err := CreateCondition(json)
	if err == nil {
		t.Error("CreateCondition should fail for invalid ref type")
	}
}

func TestCreateCondition_MissingRefName(t *testing.T) {
	json := gjson.Parse(`{"type":"equals","value1":{"type":"request_header"},"value2":"abc"}`)
	_, err := CreateCondition(json)
	if err == nil {
		t.Error("CreateCondition should fail for missing ref name")
	}
}

// --- ConditionSet tests ---
func TestConditionSet_Matches_AllMatch(t *testing.T) {
	json := gjson.Parse(`{"conditions":[{"type":"equals","value1":{"type":"request_header","name":"x-test"},"value2":"abc"},{"type":"prefix","value":{"type":"request_query","name":"foo"},"prefix":"bar"}]}`)
	var set ConditionSet
	if err := set.FromJson(json); err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-test": {"abc"}})
	ctx.SetRequestQueries(map[string][]string{"foo": {"barbaz"}})
	if !set.Matches(ctx) {
		t.Error("ConditionSet should match when all conditions match")
	}
}

func TestConditionSet_Matches_OneNoMatch(t *testing.T) {
	json := gjson.Parse(`{"conditions":[{"type":"equals","value1":{"type":"request_header","name":"x-test"},"value2":"abc"},{"type":"prefix","value":{"type":"request_query","name":"foo"},"prefix":"bar"}]}`)
	var set ConditionSet
	if err := set.FromJson(json); err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	ctx := NewEditorContext()
	ctx.SetRequestHeaders(map[string][]string{"x-test": {"abc"}})
	ctx.SetRequestQueries(map[string][]string{"foo": {"baz"}})
	if set.Matches(ctx) {
		t.Error("ConditionSet should not match if one condition does not match")
	}
}

func TestConditionSet_Matches_Empty(t *testing.T) {
	json := gjson.Parse(`{"conditions":[]}`)
	var set ConditionSet
	if err := set.FromJson(json); err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	ctx := NewEditorContext()
	if !set.Matches(ctx) {
		t.Error("ConditionSet with no conditions should always match")
	}
}

// --- GetType/GetRefs coverage ---
func TestCondition_GetTypeAndRefs(t *testing.T) {
	json := gjson.Parse(`{"type":"equals","value1":{"type":"request_header","name":"x-test"},"value2":"abc"}`)
	cond, err := CreateCondition(json)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	if cond.GetType() != "equals" {
		t.Error("GetType should return 'equals'")
	}
	refs := cond.GetRefs()
	if len(refs) != 1 || refs[0].Type != "request_header" || refs[0].Name != "x-test" {
		t.Error("GetRefs should return correct ref")
	}
}

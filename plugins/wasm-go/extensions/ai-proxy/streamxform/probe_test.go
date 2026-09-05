package streamxform

import (
	"encoding/json"
	"fmt"
	"testing"
)

// probe 只报告行为，不断言——先看清哪些形态会坏
func probe(t *testing.T, name, in string) {
	t.Helper()
	got := xform(in, 4096)
	var m map[string]any
	err := json.Unmarshal([]byte(got), &m)
	status := "合法JSON"
	if err != nil {
		status = "非法JSON: " + err.Error()
	}
	fmt.Printf("  %-22s %s\n    -> %s\n", name, status, got)
}

func TestProbeShapes(t *testing.T) {
	probe(t, "content 是数组(多模态)",
		`{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	probe(t, "messages 为空",
		`{"model":"m","messages":[]}`)
	probe(t, "message 含 tool_calls",
		`{"model":"m","messages":[{"role":"assistant","content":"x","tool_calls":[{"id":"c1","function":{"name":"f","arguments":"{}"}}]}]}`)
	probe(t, "顶层未知对象字段",
		`{"model":"m","metadata":{"k":"v"},"messages":[{"role":"user","content":"U"}]}`)
	probe(t, "顶层 tools 数组",
		`{"model":"m","tools":[{"type":"function","function":{"name":"f","description":"d"}}],"messages":[{"role":"user","content":"U"}]}`)
	probe(t, "content 为 null",
		`{"model":"m","messages":[{"role":"assistant","content":null,"tool_calls":[]}]}`)
	probe(t, "多条 user 消息",
		`{"model":"m","messages":[{"role":"user","content":"A"},{"role":"assistant","content":"B"},{"role":"user","content":"C"}]}`)
	probe(t, "无 messages 字段",
		`{"model":"m","max_tokens":10}`)
	probe(t, "key 里含转义",
		`{"model":"m","messages":[{"role":"user","content":"a\"b"}]}`)
}

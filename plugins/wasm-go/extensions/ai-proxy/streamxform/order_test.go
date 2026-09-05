package streamxform

import (
	"encoding/json"
	"strings"
	"testing"
)

func xform(in string, chunk int) string {
	tr := New()
	var sb strings.Builder
	for i := 0; i < len(in); i += chunk {
		j := i + chunk
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		sb.Write(tr.Out())
	}
	sb.Write(tr.Finish())
	return sb.String()
}

// messages 排在顶层标量之前——JSON 规范允许，客户端可任意序列化
func TestMessagesBeforeScalars(t *testing.T) {
	in := `{"messages":[{"role":"user","content":"U"}],"model":"claude-3","max_tokens":100,"stream":true}`
	got := xform(in, 4096)
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("非法 JSON: %v\n%s", err, got)
	}
	if m["model"] != "claude-3" {
		t.Errorf("model 丢了: %v  (输出 %s)", m["model"], got)
	}
	if m["max_tokens"] != float64(100) {
		t.Errorf("max_tokens 丢了: %v", m["max_tokens"])
	}
	if m["stream"] != true {
		t.Errorf("stream 丢了: %v", m["stream"])
	}
}

// message 对象内 content 排在 role 之前
func TestContentBeforeRole(t *testing.T) {
	in := `{"model":"m","messages":[{"content":"S","role":"system"},{"content":"U","role":"user"}]}`
	got := xform(in, 4096)
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("非法 JSON: %v\n%s", err, got)
	}
	if m["system"] != "S" {
		t.Errorf("system 未被提取: %v  (输出 %s)", m["system"], got)
	}
	msgs, _ := m["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages 数=%d 应为 1 (输出 %s)", len(msgs), got)
	}
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Errorf("role 丢了: %v", msgs[0])
	}
}

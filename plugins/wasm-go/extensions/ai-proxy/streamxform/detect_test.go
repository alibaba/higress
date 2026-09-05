package streamxform

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func detect(in string) (bool, string, string) {
	tr := New()
	var out []byte
	for i := 0; i < len(in); i += 4096 {
		j := i + 4096
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		out = append(out, tr.Out()...)
	}
	out = append(out, tr.Finish()...)
	bad, why := tr.Unsupported()
	return bad, why, string(out)
}

// 哪些形态必须 Bail、哪些必须放行。放行的输出与官方的等价性由差分测试负责，这里只验合法 JSON。
func TestUnsupportedDetection(t *testing.T) {
	mustBail := []struct{ name, in string }{
		{"claude_content_blocks", `{"model":"m","messages":[{"role":"user","content":"U","claude_content_blocks":[]}]}`},
		{"重复 key", `{"model":"m","model":"n","messages":[{"role":"user","content":"U"}]}`},
		{"messages 为空", `{"model":"m","messages":[]}`},
		{"无 messages", `{"model":"m"}`},
		{"无 model", `{"messages":[{"role":"user","content":"U"}]}`},
		{"role 非字符串", `{"model":"m","messages":[{"role":1,"content":"U"}]}`},
		{"max_tokens 非整数", `{"model":"m","max_tokens":1.5,"messages":[{"role":"user","content":"U"}]}`},
		{"stop 是字符串", `{"model":"m","stop":"END","messages":[{"role":"user","content":"U"}]}`},
		{"根不是对象", `[1,2]`},
		{"截断", `{"model":"m","messages":[{"role":"user","content":"U"}`},
	}
	for _, c := range mustBail {
		bad, why, _ := detect(c.in)
		if !bad {
			t.Errorf("%s: 应判定为不支持，实际放行了", c.name)
		} else {
			fmt.Printf("  ✓ %-22s 已拦截: %s\n", c.name, why)
		}
	}

	mustPass := []struct{ name, in string }{
		{"纯文本对话", `{"model":"m","messages":[{"role":"system","content":"S"},{"role":"user","content":"U"}]}`},
		{"多条消息", `{"model":"m","messages":[{"role":"user","content":"A"},{"role":"assistant","content":"B"}]}`},
		{"乱序字段", `{"messages":[{"content":"U","role":"user"}],"model":"m"}`},
		{"大 content", `{"model":"m","messages":[{"role":"user","content":"` + strings.Repeat("y", 100000) + `"}]}`},
		{"stop 数组", `{"model":"m","stop":["END"],"messages":[{"role":"user","content":"U"}]}`},
		{"Claude无对应字段", `{"model":"m","user":"u1","seed":42,"n":1,"messages":[{"role":"user","content":"U"}]}`},
		{"嵌套对象字段", `{"model":"m","metadata":{"k":"v"},"response_format":{"type":"json_object"},"messages":[{"role":"user","content":"U"}]}`},
		{"content 多模态数组", `{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hi"},{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA","detail":"high"}}]}]}`},
		{"message tool_calls", `{"model":"m","messages":[{"role":"assistant","content":"x","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{\"a\":1}"}}]}]}`},
		{"tool role 合并", `{"model":"m","messages":[{"role":"tool","tool_call_id":"c1","content":"r1"},{"role":"tool","tool_call_id":"c2","content":"r2"},{"role":"user","content":"U"}]}`},
		{"顶层 tools", `{"model":"m","tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}],"tool_choice":"auto","messages":[{"role":"user","content":"U"}]}`},
		{"未知字段", `{"model":"m","totally_unknown_x":"v","messages":[{"role":"user","content":"U","name":"n1"}]}`},
		{"content 为 null", `{"model":"m","messages":[{"role":"assistant","content":null}]}`},
		{"developer role", `{"model":"m","messages":[{"role":"developer","content":"D"},{"role":"user","content":"U"}]}`},
		{"reasoning", `{"model":"m","reasoning_effort":"high","messages":[{"role":"user","content":"U"}]}`},
	}
	for _, c := range mustPass {
		bad, why, out := detect(c.in)
		if bad {
			t.Errorf("%s: 应放行，却被判定不支持: %s", c.name, why)
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Errorf("%s: 输出非法 JSON: %v\n%s", c.name, err, out)
			continue
		}
		fmt.Printf("  ✓ %-22s 正常走流式\n", c.name)
	}
}

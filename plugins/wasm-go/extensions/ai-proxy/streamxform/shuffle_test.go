package streamxform

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// 穷举顶层字段与 message 内字段的多种排列，验证输出等价
func TestFieldOrderPermutations(t *testing.T) {
	cases := []struct{ name, in string }{
		{"规范顺序", `{"model":"m","max_tokens":50,"stream":true,"messages":[{"role":"system","content":"S"},{"role":"user","content":"U"}]}`},
		{"messages最前", `{"messages":[{"role":"system","content":"S"},{"role":"user","content":"U"}],"model":"m","max_tokens":50,"stream":true}`},
		{"messages居中", `{"model":"m","messages":[{"role":"system","content":"S"},{"role":"user","content":"U"}],"max_tokens":50,"stream":true}`},
		{"content先于role", `{"model":"m","max_tokens":50,"stream":true,"messages":[{"content":"S","role":"system"},{"content":"U","role":"user"}]}`},
		{"两者都乱", `{"messages":[{"content":"S","role":"system"},{"content":"U","role":"user"}],"stream":true,"model":"m","max_tokens":50}`},
	}
	for _, c := range cases {
		for _, cs := range []int{1, 7, 4096} {
			got := xform(c.in, cs)
			var m map[string]any
			if err := json.Unmarshal([]byte(got), &m); err != nil {
				t.Fatalf("%s chunk=%d 非法 JSON: %v\n%s", c.name, cs, err, got)
			}
			if m["model"] != "m" || m["max_tokens"] != float64(50) || m["stream"] != true {
				t.Errorf("%s chunk=%d 顶层字段错: %s", c.name, cs, got)
			}
			if m["system"] != "S" {
				t.Errorf("%s chunk=%d system=%v: %s", c.name, cs, m["system"], got)
			}
			msgs, _ := m["messages"].([]any)
			if len(msgs) != 1 {
				t.Errorf("%s chunk=%d messages 数=%d: %s", c.name, cs, len(msgs), got)
				continue
			}
			m0 := msgs[0].(map[string]any)
			if m0["role"] != "user" {
				t.Errorf("%s chunk=%d role=%v", c.name, cs, m0["role"])
			}
			txt := m0["content"]
			if txt != "U" {
				t.Errorf("%s chunk=%d text=%v", c.name, cs, txt)
			}
		}
		fmt.Printf("  %-14s ✓  %s\n", c.name, strings.TrimSpace(xform(c.in, 4096)))
	}
}

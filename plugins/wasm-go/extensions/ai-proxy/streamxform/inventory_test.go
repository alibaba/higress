package streamxform

import (
	"encoding/json"
	"fmt"
	"testing"
)

// 逐字段探测：进去了、被拦了、还是被静默丢了
func TestFieldInventory(t *testing.T) {
	fields := []struct{ name, frag string }{
		{"model", `"model":"m"`},
		{"max_tokens", `"max_tokens":50`},
		{"stream", `"stream":true`},
		{"temperature", `"temperature":0.7`},
		{"top_p", `"top_p":0.9`},
		{"stop", `"stop":["END"]`},
		{"top_k", `"top_k":40`},
		{"presence_penalty", `"presence_penalty":0.5`},
		{"frequency_penalty", `"frequency_penalty":0.3`},
		{"seed", `"seed":42`},
		{"n", `"n":2`},
		{"user", `"user":"u1"`},
		{"response_format", `"response_format":{"type":"json_object"}`},
		{"reasoning_effort", `"reasoning_effort":"high"`},
		{"thinking", `"thinking":{"type":"enabled","budget_tokens":2048}`},
		{"metadata", `"metadata":{"k":"v"}`},
		{"logprobs", `"logprobs":true`},
		{"stream_options", `"stream_options":{"include_usage":true}`},
	}
	fmt.Println("  字段                 结果")
	fmt.Println("  ------------------   ----------------------------------")
	for _, f := range fields {
		in := `{` + f.frag + `,"messages":[{"role":"user","content":"U"}]}`
		tr := New()
		tr.Write([]byte(in))
		tr.Out()
		out := string(tr.Finish())
		bad, why := tr.Unsupported()
		var m map[string]any
		json.Unmarshal([]byte(out), &m)
		switch {
		case bad:
			fmt.Printf("  %-18s   拦截 → 回落 (%s)\n", f.name, why)
		case hasAny(m, f.name, claudeName(f.name)):
			fmt.Printf("  %-18s   ✓ 已映射\n", f.name)
		default:
			fmt.Printf("  %-18s   丢弃（Claude 无此字段，与官方一致）\n", f.name)
		}
	}
}

func claudeName(n string) string {
	switch n {
	case "stop":
		return "stop_sequences"
	case "reasoning_effort":
		return "thinking"
	}
	return n
}

func hasAny(m map[string]any, names ...string) bool {
	for _, n := range names {
		if _, ok := m[n]; ok {
			return true
		}
	}
	return false
}

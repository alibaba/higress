package streamxform

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestStopMapping(t *testing.T) {
	for _, cs := range []int{1, 7, 4096} {
		in := `{"model":"m","stop":["END","STOP"],"messages":[{"role":"user","content":"U"}]}`
		got := xform(in, cs)
		var m map[string]any
		if err := json.Unmarshal([]byte(got), &m); err != nil {
			t.Fatalf("chunk=%d 非法 JSON: %v\n%s", cs, err, got)
		}
		ss, ok := m["stop_sequences"].([]any)
		if !ok || len(ss) != 2 || ss[0] != "END" || ss[1] != "STOP" {
			t.Errorf("chunk=%d stop_sequences=%v\n%s", cs, m["stop_sequences"], got)
			continue
		}
		if cs == 4096 {
			fmt.Printf("  stop 映射 ✓ %s\n", got)
		}
	}
}

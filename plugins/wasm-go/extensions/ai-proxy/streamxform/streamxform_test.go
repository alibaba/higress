package streamxform

import (
	"encoding/json"
	"strings"
	"testing"
)

// run 把输入按给定块大小切开喂入，模拟任意 TCP 分段
func run(t *testing.T, in string, chunk int) string {
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

const basic = `{"model":"claude-3","max_tokens":100,"stream":true,"messages":[{"role":"system","content":"你是助手"},{"role":"user","content":"你好"}]}`

func TestBasic(t *testing.T) {
	for _, cs := range []int{1, 3, 7, 16, 64, 4096} {
		got := run(t, basic, cs)
		var m map[string]any
		if err := json.Unmarshal([]byte(got), &m); err != nil {
			t.Fatalf("chunk=%d 输出不是合法 JSON: %v\n%s", cs, err, got)
		}
		if m["model"] != "claude-3" {
			t.Errorf("chunk=%d model=%v", cs, m["model"])
		}
		if m["system"] != "你是助手" {
			t.Errorf("chunk=%d system=%v", cs, m["system"])
		}
		msgs, _ := m["messages"].([]any)
		if len(msgs) != 1 {
			t.Fatalf("chunk=%d messages 数=%d，应为 1（system 已提取）", cs, len(msgs))
		}
		m0 := msgs[0].(map[string]any)
		if m0["role"] != "user" {
			t.Errorf("chunk=%d role=%v", cs, m0["role"])
		}
		if m0["content"] != "你好" {
			t.Errorf("chunk=%d content=%v", cs, m0["content"])
		}
	}
}

// 转义序列被切开时不能出错
func TestEscapeSplit(t *testing.T) {
	in := `{"model":"m","messages":[{"role":"user","content":"引号\"与反斜杠\\和换行\n还有ä"}]}`
	for _, cs := range []int{1, 2, 5, 13} {
		got := run(t, in, cs)
		var m map[string]any
		if err := json.Unmarshal([]byte(got), &m); err != nil {
			t.Fatalf("chunk=%d 非法 JSON: %v\n%s", cs, err, got)
		}
		msgs := m["messages"].([]any)
		want := "引号\"与反斜杠\\和换行\n还有ä"
		if got := msgs[0].(map[string]any)["content"]; got != want {
			t.Errorf("chunk=%d content=%q want=%q", cs, got, want)
		}
	}
}

// 大 content：验证内存不随长度增长（输出应边生成边取走）
func TestLargeContentStreams(t *testing.T) {
	big := strings.Repeat("y", 1<<20)
	in := `{"model":"m","messages":[{"role":"user","content":"` + big + `"}]}`
	tr := New()
	maxHeld := 0
	var total int
	for i := 0; i < len(in); i += 4096 {
		j := i + 4096
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		o := tr.Out()
		total += len(o)
		if len(o) > maxHeld {
			maxHeld = len(o)
		}
	}
	total += len(tr.Finish())
	// 提交点存在时，首次下发前会攒满 CommitBytes——这是回落窗口的代价。
	// 关键是它有上界、不随 content 长度增长。
	if maxHeld > CommitBytes+16*1024 {
		t.Errorf("单次持有 %d 字节，超过提交点上界（未流式）", maxHeld)
	}
	if total < 1<<20 {
		t.Errorf("输出总量 %d，小于输入 content", total)
	}
}

package streamxform

import (
	"fmt"
	"strings"
	"testing"
)

// 提交点之内发现不支持 → 调用方可干净回落（未发出任何字节）
func TestBailBeforeCommit(t *testing.T) {
	in := `{"model":"m","messages":[{"role":"user","content":"U","claude_content_blocks":[{"type":"text","text":"x"}]}]}`
	tr := New()
	emitted := 0
	for i := 0; i < len(in); i += 512 {
		j := i + 512
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		emitted += len(tr.Out())
	}
	emitted += len(tr.Finish())
	bad, why := tr.Unsupported()
	fmt.Printf("  小请求不支持形态: 不支持=%v 已发出=%d 字节 越过提交点=%v\n    原因: %s\n",
		bad, emitted, tr.Committed(), why)
	if !bad {
		t.Fatal("应判定不支持")
	}
	if emitted != 0 {
		t.Errorf("判定不支持后仍发出了 %d 字节，调用方无法干净回落", emitted)
	}
	if tr.Committed() {
		t.Error("小请求不应越过提交点")
	}
}

// 提交点之后才发现不支持 → 已发出字节，只能让请求失败
func TestBailAfterCommit(t *testing.T) {
	big := strings.Repeat("y", 200<<10) // 200KB，远超 CommitBytes
	in := `{"model":"m","messages":[{"role":"user","content":"` + big +
		`"},{"role":"user","content":"x","claude_content_blocks":[{"type":"text","text":"x"}]}]}`
	tr := New()
	emitted := 0
	for i := 0; i < len(in); i += 4096 {
		j := i + 4096
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		emitted += len(tr.Out())
	}
	emitted += len(tr.Finish())
	bad, why := tr.Unsupported()
	fmt.Printf("  大请求末尾不支持形态: 不支持=%v 已发出=%d 字节 越过提交点=%v\n    原因: %s\n",
		bad, emitted, tr.Committed(), why)
	if !bad {
		t.Fatal("应判定不支持")
	}
	if !tr.Committed() {
		t.Error("200KB 输入应已越过提交点")
	}
	if emitted == 0 {
		t.Error("越过提交点后应已发出过字节")
	}
}

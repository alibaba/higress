package streamxform

import (
	"fmt"
	"strings"
	"testing"
)

// 大 content 且 role 迟到——最坏情况：暂存到上限才降级
func TestMemWithLateRole(t *testing.T) {
	for _, mb := range []int{1, 4, 16} {
		big := strings.Repeat("y", mb<<20)
		// role 排在 content 之后，强制走有界暂存路径
		in := `{"messages":[{"content":"` + big + `","role":"user"}],"model":"m"}`
		tr := New()
		maxHeld, total := 0, 0
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
		fmt.Printf("  role迟到 输入%2dMB -> 输出 %.2fMB | 单次最大持有 %d 字节 (上限 %d)\n",
			mb, float64(total)/1048576, maxHeld, roleWaitCap)
		if maxHeld > roleWaitCap+8192 {
			t.Errorf("%dMB: 持有 %d 超过暂存上限，未有界", mb, maxHeld)
		}
	}
}

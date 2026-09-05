package streamxform

import (
	"strings"
	"testing"
)

// 超长 content 且 role 迟到：必须 Bail，不能猜成 user
func TestHoldOverflowBails(t *testing.T) {
	big := strings.Repeat("s", roleWaitCap+1024)
	in := `{"model":"m","messages":[{"content":"` + big + `","role":"system"}]}`
	tr := New()
	tr.Write([]byte(in))
	tr.Finish()
	bad, why := tr.Unsupported()
	if !bad {
		t.Fatal("超过 roleWaitCap 仍未见 role 时应 Bail，实际放行——会把 system 误判为 user")
	}
	if !strings.Contains(why, "content") {
		t.Errorf("Bail 原因应指向 content: %s", why)
	}
}

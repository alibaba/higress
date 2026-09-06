package streamxform

import (
	"strings"
	"testing"
)

// Via：子树内的回调全部落到子 hook，容器闭合的 OnLeave 回到发起方；子 hook 自己 Enter 的更深层也归它。
type viaHost struct {
	BaseProtocol
	sub   viaSub
	trace *[]string
}
type viaSub struct {
	BaseProtocol
	trace *[]string
}

func (p *viaHost) OnKey(t *Transformer) Action {
	*p.trace = append(*p.trace, "host.key:"+t.PathString())
	if t.Depth() == 1 && t.Last() == "sub" {
		return Probe()
	}
	return Pass()
}
func (p *viaHost) OnStart(t *Transformer, kind ValueKind) Action {
	*p.trace = append(*p.trace, "host.start:"+t.PathString())
	return Enter().Via(&p.sub)
}
func (p *viaHost) OnLeave(t *Transformer) { *p.trace = append(*p.trace, "host.leave:"+t.PathString()) }
func (s *viaSub) OnKey(t *Transformer) Action {
	*s.trace = append(*s.trace, "sub.key:"+t.PathString())
	if t.Last() == "in" {
		return Probe()
	}
	return Pass()
}
func (s *viaSub) OnElem(t *Transformer) Action {
	*s.trace = append(*s.trace, "sub.elem:"+t.PathString())
	return Pass()
}
func (s *viaSub) OnStart(t *Transformer, kind ValueKind) Action {
	*s.trace = append(*s.trace, "sub.start:"+t.PathString())
	return Enter() // 子 hook 自己 Enter：更深层仍归它
}
func (s *viaSub) OnLeave(t *Transformer) { *s.trace = append(*s.trace, "sub.leave:"+t.PathString()) }

func TestVia(t *testing.T) {
	in := `{"a":1,"sub":{"x":[1,2],"in":{"y":2}},"b":3}`
	for _, cs := range []int{1, 3, 4096} {
		var trace []string
		h := &viaHost{trace: &trace}
		h.sub.trace = &trace
		out, ok, why := feedAll(NewTransformer(h), in, cs)
		if !ok {
			t.Fatalf("chunk=%d: %s", cs, why)
		}
		if out != in {
			t.Fatalf("chunk=%d: 输出 %s", cs, out)
		}
		want := []string{
			"host.key:a", "host.key:sub", "host.start:sub",
			"sub.key:sub.x", "sub.key:sub.in", "sub.start:sub.in", "sub.key:sub.in.y", "sub.leave:sub.in",
			"host.leave:sub", "host.key:b", "host.leave:", // 最后是根对象闭合
		}
		got := strings.Join(trace, " ")
		if got != strings.Join(want, " ") {
			t.Fatalf("chunk=%d 回调序列不对:\n got  %s\n want %s", cs, got, strings.Join(want, " "))
		}
	}
}

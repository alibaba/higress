package streamxform

import (
	"encoding/json"
	"strings"
	"testing"
)

// 一个用来验证框架机制的最小协议：
// keep → Pass；drop → Skip；ren → 改名；wrap → 加壳；cap → Capture 后改写；
// late → Defer 直到 sig 到达；box → Enter（Lazy）；img → Prefix。
type probeProto struct {
	sigSeen bool
	tail    string
}

func (p *probeProto) OnKey(t *Transformer) Action {
	if t.Depth() == 1 {
		switch t.Last() {
		case "drop":
			return Skip()
		case "ren":
			return Pass().As("renamed")
		case "wrap":
			return Pass().Wrap([]byte(`{"v":`), []byte(`}`))
		case "cap":
			return Capture(64)
		case "late":
			if !p.sigSeen {
				return Defer(1 << 10)
			}
			return Pass().As("late_after_sig")
		case "sig":
			return Capture(16)
		case "box":
			return Enter().Lazy()
		case "arr":
			return Enter()
		case "img":
			return Prefix(8)
		case "inner":
			return Pass().Inner().Wrap([]byte(`"<`), []byte(`>"`))
		}
		return Pass()
	}
	if t.Depth() == 2 && t.Key(0) == "box" {
		return Skip() // box 里什么都不留 → Lazy 应整个消失
	}
	return Pass()
}
func (p *probeProto) OnElem(t *Transformer) Action               { return Pass() }
func (p *probeProto) OnStart(t *Transformer, k ValueKind) Action { return Pass() }
func (p *probeProto) OnValue(t *Transformer, raw []byte) {
	switch t.Last() {
	case "cap":
		t.W().Key("cap_x2")
		t.W().Byte('[')
		t.W().Raw(raw)
		t.W().Byte(',')
		t.W().Raw(raw)
		t.W().Byte(']')
	case "sig":
		p.sigSeen = true
		t.Release()
	}
}
func (p *probeProto) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	// 前 3 个字节是"协议头"，丢掉；剩下的原样流出
	if len(raw) < 3 {
		return Bail("太短"), 0
	}
	t.W().Key("img_body")
	return Pass().Wrap([]byte(`"`), []byte(`"`)), 3
}
func (p *probeProto) OnLeave(t *Transformer) {}
func (p *probeProto) Tail(t *Transformer) {
	if p.tail != "" {
		t.W().Key("tail")
		t.W().JSONString(p.tail)
	}
}

func runProbe(t *testing.T, in string, chunk int, tail string) (map[string]any, string) {
	tr := NewTransformer(&probeProto{tail: tail})
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
	if bad, why := tr.Unsupported(); bad {
		return nil, why
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(sb.String()), &m); err != nil {
		t.Fatalf("输出非法 JSON: %v\n%s", err, sb.String())
	}
	return m, sb.String()
}

func TestEngineActions(t *testing.T) {
	in := `{"keep":{"a":[1,"x",{"b":null}]},"drop":[1,2,{"z":3}],"ren":"r","wrap":[1,2],"cap":"c",` +
		`"late":{"deep":[true]},"sig":1,"box":{"k":"v"},"arr":[],"img":"HDRpayload","inner":"a\"b","n":-1.5e3}`
	for _, cs := range []int{1, 2, 3, 7, 4096} {
		m, out := runProbe(t, in, cs, "T")
		if m == nil {
			t.Fatalf("chunk=%d 意外回落: %s", cs, out)
		}
		want := map[string]string{
			"keep":           `{"a":[1,"x",{"b":null}]}`,
			"renamed":        `"r"`,
			"wrap":           `{"v":[1,2]}`,
			"cap_x2":         `["c","c"]`,
			"late_after_sig": `{"deep":[true]}`,
			"arr":            `[]`,
			"img_body":       `"payload"`,
			"inner":          `"<a\"b>"`,
			"n":              `-1.5e3`,
			"tail":           `"T"`,
		}
		_ = want
		if _, has := m["drop"]; has {
			t.Errorf("chunk=%d drop 应被丢弃", cs)
		}
		if _, has := m["box"]; has {
			t.Errorf("chunk=%d Lazy 的空容器应整个消失", cs)
		}
		if _, has := m["sig"]; has {
			t.Errorf("chunk=%d Capture 的值不应直接输出", cs)
		}
		if m["renamed"] != "r" || m["img_body"] != "payload" || m["inner"] != `<a"b>` || m["tail"] != "T" {
			t.Errorf("chunk=%d 输出不对: %s", cs, out)
		}
		if a, _ := m["arr"].([]any); a == nil {
			t.Errorf("chunk=%d 非 Lazy 的空容器应物化为 []: %s", cs, out)
		}
		late, _ := m["late_after_sig"].(map[string]any)
		if late == nil {
			t.Errorf("chunk=%d Defer 的值应在 sig 之后回放: %s", cs, out)
		}
		// 顺序：late 在 sig 之后被回放，因此排在 renamed/wrap 之后
		if strings.Index(out, `"late_after_sig"`) < strings.Index(out, `"wrap"`) {
			t.Errorf("chunk=%d 回放顺序不对: %s", cs, out)
		}
	}
}

func TestEngineSyntaxBails(t *testing.T) {
	// 只有派发帧里的语法错误会被发现；Pass 区域内部是字节透传，不做校验（由上游拒绝）。
	for _, in := range []string{`[1]`, `{"a":1,}`, `{"a" 1}`, `{"a":1}x`, `{"a":{"b":1}`, `{,"a":1}`, `{"a":1 "b":2}`, `{"a":}`} {
		tr := NewTransformer(&probeProto{})
		tr.Write([]byte(in))
		tr.Finish()
		if bad, _ := tr.Unsupported(); !bad {
			t.Errorf("%q 应 Bail", in)
		}
	}
}

func TestDeferOverflowBails(t *testing.T) {
	in := `{"late":"` + strings.Repeat("x", 2048) + `","sig":1}`
	tr := NewTransformer(&probeProto{})
	tr.Write([]byte(in))
	tr.Finish()
	if bad, why := tr.Unsupported(); !bad || !strings.Contains(why, "上限") {
		t.Errorf("Defer 超上限应 Bail: %v %s", bad, why)
	}
}

func TestOpenAIPassthroughShape(t *testing.T) {
	in := `{"messages":[{"role":"user","content":"U"}],"model":"gpt-4o","stream":true,"x":{"model":"inner"}}`
	tr := NewOpenAI(OpenAIOptions{MapModel: func(m string) string { return "M:" + m }, DetectStream: true, NormalizeUsage: true, CheckMessages: true})
	tr.Write([]byte(in))
	out := string(tr.Finish())
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("非法 JSON: %v %s", err, out)
	}
	if m["model"] != "M:gpt-4o" {
		t.Errorf("model 未改写: %s", out)
	}
	if so, _ := m["stream_options"].(map[string]any); so == nil || so["include_usage"] != true {
		t.Errorf("stream_options.include_usage 未补: %s", out)
	}
	if x, _ := m["x"].(map[string]any); x == nil || x["model"] != "inner" {
		t.Errorf("嵌套对象里的 model 不应被改: %s", out)
	}
	if p, ok := tr.Protocol().(Preluder); !ok || !p.Prelude().Stream || !p.Prelude().StreamSeen {
		t.Errorf("Prelude 未报告 stream")
	}
	// 集成层要用原始 model 写上下文键（响应侧的模型名、Azure 的请求路径都靠它）
	if pre := tr.Protocol().(Preluder).Prelude(); !pre.ModelSeen || pre.Model != "gpt-4o" {
		t.Errorf("Prelude 未报告原始 model: %+v", pre)
	}
}

// ---- 评审发现的回归用例 ----

// Release 在返回 Enter 的回调里被调用：回放必须发生在本帧回到安全点时，不能被子帧消费掉。
type releaseOnEnterProto struct{ probeProto }

func (p *releaseOnEnterProto) OnKey(t *Transformer) Action {
	if t.Depth() == 1 && t.Last() == "go" {
		p.sigSeen = true
		t.Release()
		return Enter()
	}
	return p.probeProto.OnKey(t)
}

func TestReleaseFromEnteringCallback(t *testing.T) {
	in := `{"late":1,"go":{"x":1},"keep":2}`
	tr := NewTransformer(&releaseOnEnterProto{})
	tr.Write([]byte(in))
	out := string(tr.Finish())
	if bad, why := tr.Unsupported(); bad {
		t.Fatalf("意外回落: %s", why)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("非法 JSON: %v %s", err, out)
	}
	if m["late_after_sig"] != float64(1) || m["keep"] != float64(2) {
		t.Errorf("Defer 项应在回到本帧后回放: %s", out)
	}
}

// 协议既不回放也不丢弃 Defer 项：引擎必须 Bail，而不是静默吞掉。
type forgetfulProto struct{ probeProto }

func (p *forgetfulProto) OnValue(t *Transformer, raw []byte) {} // 不再 Release

func TestLeftoverDeferredBails(t *testing.T) {
	tr := NewTransformer(&forgetfulProto{})
	tr.Write([]byte(`{"late":{"important":true},"sig":1,"keep":2}`))
	tr.Finish()
	if bad, why := tr.Unsupported(); !bad || !strings.Contains(why, "Defer") {
		t.Errorf("残留 Defer 项应 Bail: %v %s", bad, why)
	}
}

// 回放标量时补的分隔符不能泄漏成闭合前空白。
func TestReplayScalarNoWhitespaceLeak(t *testing.T) {
	tr := NewTransformer(&probeProto{})
	tr.Write([]byte(`{"late":1,"sig":1}`))
	out := string(tr.Finish())
	if out != `{"late_after_sig":1}` {
		t.Errorf("回放后多出空白: %q", out)
	}
}

// 缓冲超限 Bail 之后不能再把残缺数据交给协议回调。
type panicOnValueProto struct{ probeProto }

func (p *panicOnValueProto) OnValue(t *Transformer, raw []byte) {
	if t.Last() == "cap" && len(raw) < 100 {
		panic("协议拿到了截断的缓冲")
	}
}

func TestNoCallbackAfterBail(t *testing.T) {
	tr := NewTransformer(&panicOnValueProto{})
	tr.Write([]byte(`{"cap":"` + strings.Repeat("x", 100) + `"}`))
	tr.Finish()
	if bad, _ := tr.Unsupported(); !bad {
		t.Error("超过 Capture 上限应 Bail")
	}
}

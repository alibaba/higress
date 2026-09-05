package streamxform

import (
	"encoding/json"
	"errors"
	"math"
)

// OpenAI → 通义千问 DashScope 原生协议（qwenEnableCompatible=false）的流式转换。
//
// 逐行对照官方 qwen.go buildQwenTextGenerationRequest / chatMessage2QwenMessage：
//   - messages 落到 input.messages，每条消息 role / name / reasoning_content / tool_calls 照搬，
//     content 字符串直通，数组按 text / image 拆成 DashScope 的多模态形态（图片 URL 直通，不解码）；
//   - 顶层标量归并进 parameters，在 Tail 用官方同构 struct 一次写出（top_p 钳制、incremental_output 依赖 tools）；
//   - 请求头（Accept / X-DashScope-SSE）与路径（qwen-vl 走多模态接口）依赖 model 与 stream，集成层在放行前施加。

type QwenNativeOptions struct {
	// MapModel 复刻官方 mapModel：model 为空或映射结果为空时返回错误。
	MapModel func(model string) (string, error)
	// SupportsPreserveThinking 复刻 qwenSupportsPreserveThinking（作用于映射后的 model）。
	SupportsPreserveThinking func(model string) bool
	// EnableSearch 对应配置 qwenEnableSearch。
	EnableSearch bool
}

const (
	qwenResultFormatMessage = "message"
	qwenTopPMin             = 0.000001
	qwenTopPMax             = 0.999999
)

// 与官方逐字段对齐（只用于 Tail / 已 Capture 的小值）
type qwenParameters struct {
	ResultFormat      string    `json:"result_format,omitempty"`
	MaxTokens         int       `json:"max_tokens,omitempty"`
	RepetitionPenalty float64   `json:"repetition_penalty,omitempty"`
	N                 int       `json:"n,omitempty"`
	Seed              int       `json:"seed,omitempty"`
	Temperature       float64   `json:"temperature,omitempty"`
	TopP              float64   `json:"top_p,omitempty"`
	IncrementalOutput bool      `json:"incremental_output,omitempty"`
	EnableSearch      bool      `json:"enable_search,omitempty"`
	PreserveThinking  bool      `json:"preserve_thinking,omitempty"`
	Tools             []oaiTool `json:"tools,omitempty"`
}
type qwenToolCall struct {
	Index            int             `json:"index"`
	Id               string          `json:"id,omitempty"`
	Type             string          `json:"type"`
	Function         oaiFunctionCall `json:"function"`
	ThoughtSignature string          `json:"thought_signature,omitempty"`
	ExtraContent     map[string]any  `json:"extra_content,omitempty"`
}

type qwenPart struct {
	typ     string
	typSeen bool
	dead    bool
	urlSeen bool
}

type qwenMsg struct {
	roleSeen       bool
	contentSeen    bool
	contentWritten bool
	finalizing     bool
	part           qwenPart
}

type qwenProto struct {
	opt QwenNativeOptions

	model      string
	modelSeen  bool
	mapped     string
	stream     bool
	streamSeen bool
	maxTok, n  int
	seed       int
	temp, topP float64
	toolsRaw   []byte
	toolsN     int // -1 = tools 为 null 或未出现

	messagesSeen  bool
	inputMsgs     int
	reasoningSeen bool
	m             qwenMsg
}

// NewQwenNative 构造 OpenAI → DashScope 原生协议转换器。
func NewQwenNative(opt QwenNativeOptions) *Transformer {
	if opt.MapModel == nil {
		opt.MapModel = func(m string) (string, error) {
			if m == "" {
				return "", errors.New("missing model in request")
			}
			return m, nil
		}
	}
	if opt.SupportsPreserveThinking == nil {
		opt.SupportsPreserveThinking = func(string) bool { return false }
	}
	p := &qwenProto{opt: opt, toolsN: -1}
	t := NewTransformer(p)
	t.DupKeyBail = true
	return t
}

func (p *qwenProto) Prelude() Prelude {
	return Prelude{Model: p.model, ModelSeen: p.modelSeen, Stream: p.stream, StreamSeen: p.streamSeen}
}

// IncrementalOutput 复刻官方 parameters.incremental_output = streaming && 无 tools。
// 集成层在整份 body 结束后用它写 incrementalStreaming 上下文键（响应侧要用）。
func (p *qwenProto) IncrementalOutput() bool { return p.stream && p.toolsN <= 0 }

// ---- 派发 ----

func (p *qwenProto) OnKey(t *Transformer) Action {
	switch t.Depth() {
	case 1:
		switch t.Last() {
		case "model":
			return Capture(4 << 10)
		case "messages":
			return Probe()
		case "stream":
			return Capture(16)
		case "max_tokens", "n", "seed", "temperature", "top_p":
			return Capture(64)
		case "tools":
			return Capture(toolsCap)
		}
		return Skip() // 官方 qwenTextGenParameters 不读其余字段
	case 3:
		m := &p.m
		switch t.Last() {
		case "role":
			return Capture(256)
		case "name":
			return Capture(4 << 10)
		case "reasoning_content":
			return Probe()
		case "tool_calls":
			return Capture(assistantWaitCap)
		case "content":
			m.contentSeen = true
			return Probe()
		}
		return Skip() // tool_call_id / audio / refusal …：qwenMessage 没有这些字段
	case 5:
		pt := &p.m.part
		if pt.dead {
			return Skip()
		}
		switch t.Last() {
		case "type":
			return Capture(256)
		case "text":
			if !pt.typSeen {
				return Defer(partWaitCap)
			}
			if pt.typ != "text" {
				return Skip()
			}
			return Probe()
		case "image_url":
			if !pt.typSeen {
				return Defer(partWaitCap)
			}
			if pt.typ != "image_url" {
				return Skip()
			}
			return Probe()
		case "input_audio", "file":
			if !pt.typSeen {
				return Defer(smallCap)
			}
			if pt.typ != t.Last() {
				return Skip()
			}
			return Capture(smallCap)
		}
		return Skip()
	case 6:
		if t.Last() == "url" {
			p.m.part.urlSeen = true
			return Probe()
		}
		return Skip() // detail
	}
	return Bail("意外的路径: " + t.PathString())
}

func (p *qwenProto) OnElem(t *Transformer) Action {
	switch t.Depth() {
	case 2, 4:
		return Probe()
	}
	return Bail("意外的数组: " + t.PathString())
}

func (p *qwenProto) OnStart(t *Transformer, kind ValueKind) Action {
	w := t.W()
	switch t.Depth() {
	case 1: // messages → input.messages
		if kind != KindArray {
			return Bail("messages 不是数组")
		}
		w.PushObj("input")
		w.PushArr("messages")
		return Enter().Flat()
	case 2:
		if kind != KindObject {
			return Bail("message 不是对象")
		}
		p.m = qwenMsg{}
		p.inputMsgs++
		return Enter() // 每条消息都输出（system 不搬家）
	case 3:
		switch t.Last() {
		case "content":
			switch kind {
			case KindString:
				p.m.contentWritten = true
				return Pass() // 字符串原样直通
			case KindArray:
				p.m.contentWritten = true
				return Enter().Lazy() // 多模态：逐 part 映射；一个 part 都没落下时官方是 null，OnLeave 处理
			}
			return Skip() // 对象 / 标量 / null：ParseContent 得 nil → "content":null
		case "reasoning_content":
			if kind != KindString {
				return Bail("reasoning_content 不是字符串，官方 struct 解析失败")
			}
			return Prefix(1) // 非空才有意义（omitempty），且要记下"见过非空"
		}
	case 4:
		if kind != KindObject {
			return Skip() // 官方 ParseContent 跳过非 map 元素
		}
		p.m.part = qwenPart{}
		return Enter().Lazy()
	case 5:
		pt := &p.m.part
		switch t.Last() {
		case "text":
			if kind != KindString {
				pt.dead = true
				return Skip()
			}
			return Prefix(1)
		case "image_url":
			if kind != KindObject {
				pt.dead = true
				return Skip()
			}
			return Enter().Flat()
		}
	case 6: // image_url.url → "image"
		if kind != KindString {
			return Bail("image_url.url 不是字符串，官方会 panic")
		}
		return Prefix(1)
	}
	_ = w
	return Bail("意外的 Probe: " + t.PathString())
}

// ---- 值到齐 ----

func (p *qwenProto) OnValue(t *Transformer, raw []byte) {
	w := t.W()
	isNull := string(raw) == "null"
	switch t.Depth() {
	case 1:
		switch t.Last() {
		case "model":
			s, ok := jsonUnquote(raw)
			if !ok {
				t.Bail("model 不是字符串")
				return
			}
			p.model, p.modelSeen = s, true
			mapped, err := p.opt.MapModel(s)
			if err != nil {
				t.Bail(err.Error())
				return
			}
			p.mapped = mapped
		case "stream":
			switch string(raw) {
			case "true":
				p.stream = true
			case "false", "null":
			default:
				t.Bail("stream 不是布尔")
				return
			}
			p.streamSeen = !isNull
		case "max_tokens", "n", "seed":
			if isNull {
				return
			}
			if !isIntLiteral(raw) {
				t.Bail(t.Last() + " 不是整数")
				return
			}
			v := atoi(raw)
			switch t.Last() {
			case "max_tokens":
				p.maxTok = v
			case "n":
				p.n = v
			default:
				p.seed = v
			}
		case "temperature", "top_p":
			if isNull {
				return
			}
			if !isNumLiteral(raw) {
				t.Bail(t.Last() + " 不是数字")
				return
			}
			f, _ := parseFloat(raw)
			if t.Last() == "temperature" {
				p.temp = f
			} else {
				p.topP = f
			}
		case "tools":
			if isNull {
				return
			}
			var tools []oaiTool
			if err := json.Unmarshal(raw, &tools); err != nil {
				t.Bail("tools 解析失败: " + err.Error())
				return
			}
			p.toolsRaw = append([]byte(nil), raw...)
			p.toolsN = len(tools)
		}
	case 3:
		switch t.Last() {
		case "role":
			s, ok := jsonUnquote(raw)
			if !ok {
				t.Bail("role 不是字符串")
				return
			}
			p.m.roleSeen = true
			w.Key("role")
			w.JSONString(s)
		case "name":
			s, ok := jsonUnquote(raw)
			if !ok {
				t.Bail("name 不是字符串")
				return
			}
			if s != "" { // omitempty
				w.Key("name")
				w.JSONString(s)
			}
		case "tool_calls":
			if isNull {
				return
			}
			var tcs []qwenToolCall
			if err := json.Unmarshal(raw, &tcs); err != nil {
				t.Bail("tool_calls 解析失败: " + err.Error())
				return
			}
			if len(tcs) > 0 { // omitempty
				b, _ := json.Marshal(tcs)
				w.Key("tool_calls")
				w.Raw(b)
			}
		}
	case 5:
		pt := &p.m.part
		switch t.Last() {
		case "type":
			pt.typSeen = true
			s, ok := jsonUnquote(raw)
			if !ok {
				pt.dead = true
				t.DropDeferred()
				return
			}
			pt.typ = s
			switch s {
			case "text", "image_url", "input_audio", "file":
				if len(t.Deferred()) > 0 {
					t.Release()
				}
			default:
				pt.dead = true // ParseContent 的 switch 匹配不到 → 不进列表
				t.DropDeferred()
			}
		case "input_audio", "file":
			// 官方 ParseContent 会做 .(string) 断言，缺字段即 panic；成功则在 qwen 里落成空对象 {}
			var obj map[string]any
			if err := json.Unmarshal(raw, &obj); err != nil {
				pt.dead = true
				return
			}
			keys := []string{"file_id"}
			if t.Last() == "input_audio" {
				keys = []string{"data", "format"}
			}
			for _, k := range keys {
				if _, ok := obj[k].(string); !ok {
					t.Bail(t.Last() + "." + k + " 缺失，官方会 panic")
					return
				}
			}
			w.Open() // {}
		}
	}
}

func (p *qwenProto) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	w := t.W()
	empty := complete && len(raw) == 0
	switch t.Depth() {
	case 3: // reasoning_content
		if empty {
			return Skip(), 0 // omitempty
		}
		p.reasoningSeen = true
		w.KeyRaw(t.KeyRaw())
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	case 5: // part.text → {"text":…}
		if empty {
			w.Open() // {}
			return Skip(), 0
		}
		w.KeyRaw(t.KeyRaw())
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	case 6: // image_url.url → {"image":…}
		if empty {
			w.Open()
			return Skip(), 0
		}
		w.Key("image")
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	}
	return Bail("意外的 Prefix: " + t.PathString()), 0
}

// ---- 容器闭合 ----

func (p *qwenProto) OnLeave(t *Transformer) {
	w := t.W()
	switch t.Depth() {
	case 1: // messages：官方 make(...,0)，物化 [] 后成对弹出 input.messages 两层
		p.messagesSeen = true
		if p.inputMsgs == 0 {
			t.Bail("no message found in the request body")
			return
		}
		w.Open()
		w.Pop()
		w.Pop()
	case 2:
		m := &p.m
		m.finalizing = true
		if len(t.Deferred()) > 0 {
			t.ReleaseNow()
			if t.dead {
				return
			}
		}
		if !m.roleSeen { // Role 无 omitempty
			w.Key("role")
			w.RawString(`""`)
		}
		if !m.contentWritten { // ParseContent 得 nil → null；未出现同样是 nil
			w.Key("content")
			w.RawString("null")
		}
	case 3: // content 数组：官方 contents 从 nil 开始 append，一个都没落下就是 null
		if !w.Opened(w.Level()) {
			w.KeyAt(w.Level()-1, "content")
			w.RawString("null")
		}
	case 4:
		if p.m.part.dead || !p.m.part.typSeen {
			t.DropDeferred()
		}
	case 5:
		if !p.m.part.urlSeen {
			t.Bail("image_url.url 缺失，官方会 panic")
		}
	}
}

// ---- 收尾 ----

func (p *qwenProto) Tail(t *Transformer) {
	w := t.W()
	if !p.messagesSeen {
		t.Bail("no message found in the request body")
		return
	}
	if !p.modelSeen {
		mapped, err := p.opt.MapModel("")
		if err != nil {
			t.Bail(err.Error())
			return
		}
		p.mapped = mapped
	}
	w.Key("model")
	w.JSONString(p.mapped)
	params := qwenParameters{
		ResultFormat:      qwenResultFormatMessage,
		MaxTokens:         p.maxTok,
		N:                 p.n,
		Seed:              p.seed,
		Temperature:       p.temp,
		TopP:              math.Max(qwenTopPMin, math.Min(p.topP, qwenTopPMax)),
		IncrementalOutput: p.IncrementalOutput(),
		EnableSearch:      p.opt.EnableSearch,
		PreserveThinking:  p.reasoningSeen && p.opt.SupportsPreserveThinking(p.mapped),
	}
	if p.toolsRaw != nil {
		_ = json.Unmarshal(p.toolsRaw, &params.Tools) // OnValue 已校验
	}
	b, _ := json.Marshal(params)
	w.Key("parameters")
	w.Raw(b)
}

func parseFloat(raw []byte) (float64, error) {
	var f float64
	err := json.Unmarshal(raw, &f)
	return f, err
}

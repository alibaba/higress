package streamxform

import "encoding/json"

// OpenAI 兼容族的流式透传协议。
//
// 对应官方 defaultTransformRequestBody + normalizeOpenAiRequestBody + convertDeveloperRoleToSystem：
// 字节透传，只在几处动手——改写 model、必要时补 stream_options.include_usage、
// 发现 developer role 时回落（官方那条路径会把整个请求经 struct 重新序列化，流式无法复刻）。
//
// Qwen 兼容模式 / 智谱 / OpenRouter 在此之上各有一点自己的逻辑，以 Variant 手写接入
// （proto_openai_variants.go）——它们是代码，不是规则表。
type OpenAIOptions struct {
	// MapModel 复刻 getMappedModel：找不到映射就原样返回，永不出错。
	MapModel func(model string) string
	// ModelOnlyIfPresent：官方只在 model 存在时才 sjson 改写（Qwen 兼容模式）；
	// 默认路径缺失时会补一个 "model":<映射空串的结果>。
	ModelOnlyIfPresent bool
	// DetectStream：chat / videos / video_remix 需要读取 stream（副作用由集成层施加）。
	DetectStream bool
	// NormalizeUsage：chat / completion 且未禁用统计时，stream 为真则补 include_usage。
	NormalizeUsage bool
	// DeveloperRoleSupported 为 false 时见到 developer role 回落。
	DeveloperRoleSupported bool
	// CheckMessages：apiName 为 chat 时才需要进入 messages 检查 role。
	CheckMessages bool
	// Variant：provider 特有逻辑；nil = 纯默认路径。
	Variant OpenAIVariant
}

// OpenAIVariant 是 OpenAI 兼容族里某个 provider 的特有逻辑。
type OpenAIVariant interface {
	// TopKey 决定一个顶层 key 的动作；ok=false 表示交给基础协议。
	TopKey(t *Transformer, key string) (Action, bool)
	// TopValue 接收 Variant 自己 Capture 的顶层值。
	TopValue(t *Transformer, key string, raw []byte)
	// NeedReasoningScan：是否需要知道 messages[].reasoning_content 有没有非空的。
	NeedReasoningScan() bool
	// Tail 在末尾输出 provider 特有的字段。
	Tail(t *Transformer, base *OpenAIState)
}

// OpenAIState 是基础协议暴露给 Variant 的状态。
type OpenAIState struct {
	ModelSeen     bool
	Model         string // 原始 model（ModelSeen 为真时有效）
	Mapped        string // 映射后的 model（ModelSeen 为真时有效）
	ReasoningSeen bool   // 某条 message 的 reasoning_content 非空（gjson String() != ""）
}

type openaiProto struct {
	opt OpenAIOptions
	st  OpenAIState

	stream         bool
	streamSeen     bool
	streamOpts     []byte
	streamOptsSeen bool
	scanReasoning  bool
	msgRCSeen      bool     // 当前 message 里已见过 reasoning_content（gjson 只看第一个）
	movedTop       []string // 被 Capture 后挪到末尾输出的顶层 key：再出现就无法保持"后者覆盖"的顺序
}

// NewOpenAI 构造 OpenAI 兼容透传协议的转换器。
func NewOpenAI(opt OpenAIOptions) *Transformer {
	if opt.MapModel == nil {
		opt.MapModel = func(m string) string { return m }
	}
	p := &openaiProto{opt: opt}
	if opt.Variant != nil {
		p.scanReasoning = opt.Variant.NeedReasoningScan()
	}
	t := NewTransformer(p)
	t.DupKeyBail = false // sjson 语义：重复 key 只动第一个，其余原样保留
	return t
}

func (p *openaiProto) Prelude() Prelude {
	return Prelude{Model: p.st.Model, ModelSeen: p.st.ModelSeen, Stream: p.stream, StreamSeen: p.streamSeen && p.opt.DetectStream}
}

func (p *openaiProto) enterMessages() bool {
	return (p.opt.CheckMessages && !p.opt.DeveloperRoleSupported) || p.scanReasoning
}

// moved 记录一个被挪到末尾输出的顶层 key。
// sjson 对重复 key 只动第一个、其余原样保留，输出里后者仍在后面；
// 而我们把第一个挪到末尾后，"后者覆盖前者"的顺序就反了——这种输入只能回落。
func (p *openaiProto) moved(t *Transformer, key string, a Action) Action {
	if a.kind == akCapture {
		p.movedTop = append(p.movedTop, key)
	}
	return a
}

func (p *openaiProto) OnKey(t *Transformer) Action {
	switch t.Depth() {
	case 1:
		k := t.Last()
		for _, m := range p.movedTop {
			if m == k {
				return Bail("顶层重复 key " + k + " 已被提前捕获，无法保持后者覆盖的顺序")
			}
		}
		if p.opt.Variant != nil {
			if a, ok := p.opt.Variant.TopKey(t, k); ok {
				return p.moved(t, k, a)
			}
		}
		switch t.Last() {
		case "model":
			if p.st.ModelSeen {
				return Pass() // sjson 只改第一个
			}
			return Capture(4 << 10)
		case "stream":
			// 副作用（Accept / isStreaming）要 DetectStream，include_usage 的判定要 NormalizeUsage；
			// 两者任一需要就得看一眼 stream 的值。
			if (!p.opt.DetectStream && !p.opt.NormalizeUsage) || p.streamSeen {
				return Pass()
			}
			return Observe(256)
		case "stream_options":
			if !p.opt.NormalizeUsage || p.streamOptsSeen {
				return Pass()
			}
			return p.moved(t, k, Capture(16<<10))
		case "messages":
			if p.enterMessages() {
				if p.scanReasoning {
					return Probe() // gjson 对非数组的 Array() 语义特殊，非数组回落
				}
				return Enter().Lenient()
			}
			return Pass()
		}
		return Pass()
	case 3:
		switch t.Last() {
		case "role":
			if p.opt.CheckMessages && !p.opt.DeveloperRoleSupported {
				return Observe(256)
			}
		case "reasoning_content":
			if p.scanReasoning && !p.msgRCSeen {
				p.msgRCSeen = true
				return Probe()
			}
		}
		return Pass()
	}
	return Pass()
}

func (p *openaiProto) OnElem(t *Transformer) Action {
	if t.Depth() == 2 {
		p.msgRCSeen = false
		return Enter().Lenient()
	}
	return Pass()
}

func (p *openaiProto) OnStart(t *Transformer, kind ValueKind) Action {
	switch t.Depth() {
	case 1: // messages
		if kind != KindArray {
			return Bail("messages 不是数组，gjson Array() 语义未复刻")
		}
		return Enter()
	case 3: // reasoning_content：只需要知道是否非空，不缓冲
		switch kind {
		case KindString:
			return Prefix(1)
		case KindNull, KindBool, KindNumber:
			return Observe(64)
		default:
			p.st.ReasoningSeen = true // 对象/数组的 String() 是原始文本，非空
			return Pass()
		}
	}
	return Pass()
}

func (p *openaiProto) OnValue(t *Transformer, raw []byte) {
	switch t.Depth() {
	case 1:
		k := t.Last()
		if p.opt.Variant != nil {
			switch k {
			case "model", "stream", "stream_options":
			default:
				p.opt.Variant.TopValue(t, k, raw)
				return
			}
		}
		switch k {
		case "model":
			// gjson.String() 对非字符串会给出字符串表示再被 sjson 写回成字符串——类型会变。
			// 这种输入极罕见，回落交给官方处理，不在这里复刻。
			s, ok := jsonUnquote(raw)
			if !ok {
				t.Bail("model 不是字符串")
				return
			}
			p.st.ModelSeen = true
			p.st.Model = s
			p.st.Mapped = p.opt.MapModel(s)
			w := t.W()
			w.KeyRaw(t.KeyRaw()) // 保留原文的 "model": 写法，与 sjson 原地改写一致
			w.JSONString(p.st.Mapped)
		case "stream":
			p.streamSeen = true
			p.stream = gjsonBool(raw)
		case "stream_options":
			p.streamOptsSeen = true
			p.streamOpts = append([]byte(nil), raw...)
		}
	case 3:
		switch t.Last() {
		case "role":
			if s, ok := jsonUnquote(raw); ok && s == "developer" {
				t.Bail("developer role：官方会把整个请求经 struct 重新序列化，流式无法等价复刻")
			}
		case "reasoning_content":
			if string(raw) != "null" {
				p.st.ReasoningSeen = true
			}
		}
	}
}

func (p *openaiProto) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	if t.Depth() == 3 && t.Last() == "reasoning_content" {
		if !(complete && len(raw) == 0) {
			p.st.ReasoningSeen = true
		}
		t.W().KeyRaw(t.KeyRaw())
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	}
	return Bail("意外的 Prefix: " + t.PathString()), 0
}

func (p *openaiProto) OnLeave(t *Transformer) {}

func (p *openaiProto) Tail(t *Transformer) {
	w := t.W()
	if !p.st.ModelSeen && !p.opt.ModelOnlyIfPresent {
		// sjson.SetBytes 在 model 缺失时会补一个（映射空串的结果）
		w.Key("model")
		w.JSONString(p.opt.MapModel(""))
	}
	if p.opt.Variant != nil {
		p.opt.Variant.Tail(t, &p.st)
		if t.dead {
			return
		}
	}
	if !p.opt.NormalizeUsage {
		return
	}
	if !p.stream {
		if p.streamOptsSeen {
			w.Key("stream_options")
			w.Raw(p.streamOpts)
		}
		return
	}
	if !p.streamOptsSeen {
		w.Key("stream_options")
		w.RawString(`{"include_usage":true}`)
		return
	}
	// 已有 stream_options：只在缺 include_usage 时追加（gjson Exists 语义）
	var m map[string]json.RawMessage
	if err := json.Unmarshal(p.streamOpts, &m); err != nil || m == nil {
		t.Bail("stream_options 不是对象，sjson 的处理方式未复刻")
		return
	}
	w.Key("stream_options")
	if _, has := m["include_usage"]; has {
		w.Raw(p.streamOpts)
		return
	}
	body := p.streamOpts[:len(p.streamOpts)-1] // 去掉 }
	w.Raw(body)
	if len(m) > 0 {
		w.Byte(',')
	}
	w.RawString(`"include_usage":true}`)
}

// gjsonStringNonEmpty 复刻 gjson.Result.String() != ""：只有 null 与 "" 是空。
func gjsonStringNonEmpty(raw []byte) bool {
	return len(raw) > 0 && string(raw) != "null" && string(raw) != `""`
}

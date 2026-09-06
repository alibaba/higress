package streamxform

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

// OpenAI → Gemini 的流式转换协议。
//
// 逐行对照官方 gemini.go buildGeminiChatRequest。结构差异比 Claude 大：
//   - messages → contents，assistant → model，system 整条提到顶层 system_instruction；
//   - 每条 content 变成 parts：字符串 → [{text}]，image_url → inlineData（data: URL 拆前缀后主体直通）；
//   - 一堆顶层标量归并进 generationConfig，在 Tail 一次写出；
//   - 请求路径依赖 model 与 stream（集成层在放行请求头前施加）。
//
// 官方对 http(s) 图片会异步抓取再内联，流式无法复刻——遇到就回落。

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiOptions struct {
	// MapModel 复刻官方 mapModel：model 为空或映射结果为空时返回错误。
	MapModel func(model string) (string, error)
	// ThinkingModel 复刻 geminiThinkingModels[映射后的 model]。
	ThinkingModel func(mapped string) bool
	// ThinkingBudget 对应配置 geminiThinkingBudget。
	ThinkingBudget int64
	// SafetySettings 对应配置 geminiSafetySetting（官方按 map 遍历，顺序本就不定）。
	SafetySettings []GeminiSafetySetting
}

// 与官方逐字段对齐的输出结构（只用于 Tail / 已 Capture 的小值）
type geminiGenerationConfig struct {
	Temperature        float64               `json:"temperature,omitempty"`
	TopP               float64               `json:"topP,omitempty"`
	TopK               int64                 `json:"topK,omitempty"`
	Seed               int64                 `json:"seed,omitempty"`
	Logprobs           bool                  `json:"logprobs,omitempty"`
	MaxOutputTokens    int                   `json:"maxOutputTokens,omitempty"`
	CandidateCount     int                   `json:"candidateCount,omitempty"`
	StopSequences      []string              `json:"stopSequences,omitempty"`
	PresencePenalty    int64                 `json:"presencePenalty,omitempty"`
	FrequencyPenalty   int64                 `json:"frequencyPenalty,omitempty"`
	ResponseModalities []string              `json:"responseModalities,omitempty"`
	NegativePrompt     string                `json:"negativePrompt,omitempty"`
	ThinkingConfig     *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
	MediaResolution    string                `json:"mediaResolution,omitempty"`
}
type geminiThinkingConfig struct {
	IncludeThoughts bool  `json:"includeThoughts,omitempty"`
	ThinkingBudget  int64 `json:"thinkingBudget,omitempty"`
}
type geminiTools struct {
	FunctionDeclarations any `json:"function_declarations,omitempty"`
}
type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}
type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type gemPart struct {
	typ     string
	typSeen bool
	dead    bool
	urlSeen bool
}

type gemMsg struct {
	role         string
	roleSeen     bool
	roleWritten  bool
	contentSeen  bool
	partsWritten bool
	finalizing   bool
	part         gemPart
}

type geminiProto struct {
	opt GeminiOptions

	model      string
	modelSeen  bool
	stream     bool
	streamSeen bool
	temp, topP float64
	maxTok     int
	presence   int64
	frequency  int64
	logprobs   bool
	modalities []string
	tools      ToolsHook

	messagesSeen bool
	inputMsgs    int
	sysSeen      bool
	sysParts     []byte // 已序列化的 parts 数组
	m            gemMsg
}

// NewGemini 构造 OpenAI → Gemini 转换器。
func NewGemini(opt GeminiOptions) *Transformer {
	if opt.MapModel == nil {
		opt.MapModel = func(m string) (string, error) {
			if m == "" {
				return "", errors.New("missing model in request")
			}
			return m, nil
		}
	}
	if opt.ThinkingModel == nil {
		opt.ThinkingModel = func(string) bool { return false }
	}
	p := &geminiProto{opt: opt, tools: ToolsHook{ParamsKey: "parameters"}}
	t := NewTransformer(p)
	t.DupKeyBail = true
	return t
}

func (p *geminiProto) Prelude() Prelude {
	return Prelude{Model: p.model, ModelSeen: p.modelSeen, Stream: p.stream, StreamSeen: p.streamSeen}
}

// ---- 派发 ----

func (p *geminiProto) OnKey(t *Transformer) Action {
	switch t.Depth() {
	case 1:
		switch t.Last() {
		case "model":
			return Capture(4 << 10)
		case "messages":
			return Probe()
		case "stream", "logprobs":
			return Capture(16)
		case "temperature", "top_p", "max_tokens", "presence_penalty", "frequency_penalty":
			return Capture(64)
		case "modalities":
			return Capture(4 << 10)
		case "tools":
			return Probe() // 逐元素流式
		}
		return Skip() // stop / seed / n / max_completion_tokens / tool_choice …：官方不读
	case 3:
		m := &p.m
		switch t.Last() {
		case "role":
			return Capture(256)
		case "content":
			if !m.roleSeen {
				return Defer(roleWaitCap)
			}
			m.contentSeen = true
			if m.role == "system" {
				return Capture(systemCap) // 整体提到 system_instruction；上限对齐官方的请求体上限
			}
			return Probe()
		}
		return Skip() // tool_calls / name / …：官方不读
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
		}
		return Skip()
	case 6:
		if t.Last() == "url" {
			p.m.part.urlSeen = true
			return Prefix(urlPrefixWin)
		}
		return Skip()
	}
	return Bail("意外的路径: " + t.PathString())
}

func (p *geminiProto) OnElem(t *Transformer) Action {
	switch t.Depth() {
	case 2, 4:
		return Probe()
	}
	return Bail("意外的数组: " + t.PathString())
}

func (p *geminiProto) OnStart(t *Transformer, kind ValueKind) Action {
	w := t.W()
	if t.Depth() == 1 && t.Last() == "tools" { // tools → tools:[{function_declarations:[...]}]（官方 Tools != nil 即物化，空数组也输出）
		switch kind {
		case KindNull:
			return Skip()
		case KindArray:
			w.PushArr("tools")
			w.PushObj("")
			w.PushArr("function_declarations")
			return Enter().Flat().Via(&p.tools) // 内部交给子 hook；闭合回到这里 Pop
		}
		return Bail("tools 不是数组，官方 struct 解析失败")
	}
	switch t.Depth() {
	case 1: // messages → contents（官方 make(…,0)：全是 system 时也输出 []）
		if kind != KindArray {
			return Bail("messages 不是数组")
		}
		return Enter().As("contents")
	case 2:
		if kind != KindObject {
			return Bail("message 不是对象")
		}
		p.m = gemMsg{}
		p.inputMsgs++
		return Enter().Lazy() // system 消息不产生元素
	case 3: // content（非 system，role 已知）
		switch kind {
		case KindString:
			p.writeRole(t)
			p.m.partsWritten = true
			return Prefix(1) // 官方 Text 带 omitempty：空串是 [{}]，得先看一眼
		case KindArray:
			p.writeRole(t)
			p.m.partsWritten = true
			return Enter().As("parts")
		}
		return Skip() // 对象 / 标量 / null：ParseContent 得空 → "parts":[]
	case 4:
		if kind != KindObject {
			return Skip()
		}
		p.m.part = gemPart{}
		return Enter().Lazy()
	case 5:
		pt := &p.m.part
		switch t.Last() {
		case "text":
			if kind != KindString {
				pt.dead = true
				return Skip()
			}
			return Prefix(1) // 空串 → {}（Text omitempty）
		case "image_url":
			if kind != KindObject {
				pt.dead = true
				return Skip()
			}
			return Enter().Flat()
		}
	}
	_ = w
	return Bail("意外的 Probe: " + t.PathString())
}

// ---- 值到齐 ----

func (p *geminiProto) OnValue(t *Transformer, raw []byte) {
	switch t.Depth() {
	case 1:
		p.topValue(t, raw)
	case 3:
		p.msgValue(t, raw)
	case 5:
		p.partValue(t, raw)
	}
}

func (p *geminiProto) topValue(t *Transformer, raw []byte) {
	isNull := string(raw) == "null"
	switch t.Last() {
	case "model":
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("model 不是字符串")
			return
		}
		p.model, p.modelSeen = s, true
	case "stream", "logprobs":
		switch string(raw) {
		case "true":
			if t.Last() == "stream" {
				p.stream = true
			} else {
				p.logprobs = true
			}
		case "false", "null":
		default:
			t.Bail(t.Last() + " 不是布尔")
			return
		}
		if t.Last() == "stream" {
			p.streamSeen = !isNull
		}
	case "temperature", "top_p", "presence_penalty", "frequency_penalty":
		if isNull {
			return
		}
		f, err := strconv.ParseFloat(string(raw), 64)
		if err != nil || !isNumLiteral(raw) {
			t.Bail(t.Last() + " 不是数字")
			return
		}
		switch t.Last() {
		case "temperature":
			p.temp = f
		case "top_p":
			p.topP = f
		case "presence_penalty":
			p.presence = int64(f) // 官方 int64(float64)：截断
		default:
			p.frequency = int64(f)
		}
	case "max_tokens":
		if isNull {
			return
		}
		if !isIntLiteral(raw) {
			t.Bail("max_tokens 不是整数")
			return
		}
		p.maxTok = atoi(raw)
	case "modalities":
		if isNull {
			return
		}
		var ss []string
		if err := json.Unmarshal(raw, &ss); err != nil {
			t.Bail("modalities 不是字符串数组")
			return
		}
		p.modalities = ss
	}
}

func (p *geminiProto) msgValue(t *Transformer, raw []byte) {
	m := &p.m
	switch t.Last() {
	case "role":
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("role 不是字符串")
			return
		}
		m.role, m.roleSeen = s, true
		if len(t.Deferred()) > 0 {
			t.Release()
		}
	case "content": // 只有 system 的 content 会 Capture 到这里
		parts, err := geminiPartsFromContent(raw)
		if err != nil {
			t.Bail(err.Error())
			return
		}
		p.sysSeen, p.sysParts = true, parts
	}
}

func (p *geminiProto) partValue(t *Transformer, raw []byte) {
	pt := &p.m.part
	if t.Last() != "type" {
		return
	}
	pt.typSeen = true
	s, ok := jsonUnquote(raw)
	if !ok {
		pt.dead = true
		t.DropDeferred()
		return
	}
	pt.typ = s
	switch s {
	case "text", "image_url":
		if len(t.Deferred()) > 0 {
			t.Release()
		}
	default:
		pt.dead = true // input_audio / file / 未知：官方 gemini 分支跳过
		t.DropDeferred()
	}
}

// OnPrefix：image_url.url 的前缀窗口。复刻官方 handleContentTypeImageUrl + baseStr2InlineData。
func (p *geminiProto) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	w := t.W()
	switch t.Depth() {
	case 3: // 字符串 content → parts:[{text}]；空串 → [{}]
		if complete && len(raw) == 0 {
			w.Key("parts")
			w.RawString(`[{}]`)
			return Skip(), 0
		}
		w.Key("parts")
		return Pass().Wrap([]byte(`[{"text":"`), []byte(`"}]`)), 0
	case 5: // text part
		if complete && len(raw) == 0 {
			w.Open() // {}
			return Skip(), 0
		}
		w.KeyRaw(t.KeyRaw())
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	}
	dec, off := unescapePrefix(raw)
	if isHTTPURLPrefix(dec) {
		return Bail("http(s) 图片官方会异步抓取后内联，流式无法复刻"), 0
	}
	if !bytes.HasPrefix(dec, []byte("data:")) {
		if !complete {
			return Bail("非 data: 的图片串超出前缀窗口"), 0
		}
		// 官方 baseStr2InlineData：记错误日志，给一个空的 inlineData
		w.Key("inlineData")
		w.RawString(`{"mimeType":"","data":""}`)
		return Skip(), 0
	}
	semi := bytes.IndexByte(dec, ';')
	if semi < 0 {
		if !complete {
			return Bail("data URL 头超出前缀窗口"), 0
		}
		w.Open() // 官方：拆分失败返回 nil → 空 part {}
		return Skip(), 0
	}
	mime := string(dec[5:semi])
	rest := dec[semi+1:]
	resumeDec := semi + 1
	if !complete && len(rest) < len("base64,") {
		return Bail("data URL 头在窗口边界被截断"), 0
	}
	if bytes.HasPrefix(rest, []byte("base64,")) {
		resumeDec += len("base64,")
	}
	resume := off[resumeDec]
	if !complete && resume >= len(raw) {
		return Bail("data URL 数据段在窗口边界被截断"), 0
	}
	w.Key("inlineData")
	w.RawString(`{"mimeType":`)
	w.JSONString(mime)
	w.RawString(`,"data":"`)
	return Pass().Wrap(nil, []byte(`"}`)), resume
}

// ---- 容器闭合 ----

func (p *geminiProto) OnLeave(t *Transformer) {
	w := t.W()
	if t.Depth() == 1 && t.Last() == "tools" {
		w.Open() // 空 tools 也物化 [{"function_declarations":[]}]
		w.Pop()
		w.Pop()
		w.Pop()
		return
	}
	switch t.Depth() {
	case 1:
		p.messagesSeen = true
		if p.inputMsgs == 0 {
			t.Bail("no message found in the request body")
		}
	case 2:
		p.finishMessage(t)
	case 3: // parts 数组：官方总是物化
		w.Open()
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

func (p *geminiProto) writeRole(t *Transformer) {
	m := &p.m
	if m.roleWritten {
		return
	}
	m.roleWritten = true
	w := t.W()
	w.Open() // 即便 role 为空（omitempty）也要有这个元素
	role := m.role
	if role == "assistant" {
		role = "model"
	}
	if role != "" {
		w.Key("role")
		w.JSONString(role)
	}
}

func (p *geminiProto) finishMessage(t *Transformer) {
	m := &p.m
	w := t.W()
	if !m.roleSeen {
		m.role, m.roleSeen = "", true
	}
	m.finalizing = true
	if m.role == "system" {
		if len(t.Deferred()) > 0 {
			t.ReleaseNow()
			if t.dead {
				return
			}
		}
		if !m.contentSeen {
			p.sysSeen, p.sysParts = true, []byte("[]")
		}
		return
	}
	if len(t.Deferred()) > 0 {
		t.ReleaseNow()
		if t.dead {
			return
		}
	}
	p.writeRole(t)
	if !m.partsWritten {
		w.Key("parts")
		w.RawString("[]")
	}
}

// ---- 收尾 ----

func (p *geminiProto) Tail(t *Transformer) {
	w := t.W()
	if !p.messagesSeen {
		t.Bail("no message found in the request body")
		return
	}
	mapped, err := p.opt.MapModel(p.model)
	if err != nil {
		t.Bail(err.Error())
		return
	}
	if p.sysSeen {
		w.Key("system_instruction")
		w.RawString(`{"parts":`)
		w.Raw(p.sysParts)
		w.Byte('}')
	}
	if len(p.opt.SafetySettings) > 0 {
		b, _ := json.Marshal(p.opt.SafetySettings)
		w.Key("safetySettings")
		w.Raw(b)
	}
	cfg := geminiGenerationConfig{
		Temperature:        p.temp,
		TopP:               p.topP,
		MaxOutputTokens:    p.maxTok,
		PresencePenalty:    p.presence,
		FrequencyPenalty:   p.frequency,
		Logprobs:           p.logprobs,
		ResponseModalities: p.modalities,
	}
	if p.opt.ThinkingModel(mapped) {
		cfg.ThinkingConfig = &geminiThinkingConfig{IncludeThoughts: true, ThinkingBudget: p.opt.ThinkingBudget}
	}
	b, _ := json.Marshal(cfg)
	w.Key("generationConfig")
	w.Raw(b)
}

// geminiPartsFromContent 复刻 ParseContent + gemini 的 part 映射（用于已 Capture 的 system content）。
func geminiPartsFromContent(raw []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errors.New("system content 非法")
	}
	parts := make([]geminiPart, 0)
	switch c := v.(type) {
	case string:
		parts = append(parts, geminiPart{Text: c})
	case []interface{}:
		for _, it := range c {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			switch m["type"] {
			case "text":
				if s, ok := m["text"].(string); ok {
					parts = append(parts, geminiPart{Text: s})
				}
			case "image_url":
				sub, ok := m["image_url"].(map[string]interface{})
				if !ok {
					continue
				}
				u, ok := sub["url"].(string)
				if !ok {
					return nil, errors.New("image_url.url 缺失，官方会 panic")
				}
				if isHTTPURLPrefix([]byte(u)) {
					// 官方只对 contents 里的 http 图片做抓取（countImageUrl 不扫 system_instruction），
					// system 里的原样保留为 {mimeType:"", data:url}
					parts = append(parts, geminiPart{InlineData: &geminiInlineData{Data: u}})
					continue
				}
				parts = append(parts, geminiPart{InlineData: inlineDataFromDataURL(u)})
			case "input_audio":
				sub, ok := m["input_audio"].(map[string]interface{})
				if ok {
					if _, ok := sub["data"].(string); !ok {
						return nil, errors.New("input_audio.data 缺失，官方会 panic")
					}
					if _, ok := sub["format"].(string); !ok {
						return nil, errors.New("input_audio.format 缺失，官方会 panic")
					}
				}
			case "file":
				sub, ok := m["file"].(map[string]interface{})
				if ok {
					if _, ok := sub["file_id"].(string); !ok {
						return nil, errors.New("file.file_id 缺失，官方会 panic")
					}
				}
			}
		}
	}
	return json.Marshal(parts)
}

// inlineDataFromDataURL 复刻 baseStr2InlineData：拆不开返回 nil（→ 空 part）。
func inlineDataFromDataURL(s string) *geminiInlineData {
	if len(s) >= 5 && s[:5] == "data:" {
		semi := -1
		for i := 0; i < len(s); i++ {
			if s[i] == ';' {
				semi = i
				break
			}
		}
		if semi < 0 {
			return nil
		}
		mime := s[5:semi]
		data := s[semi+1:]
		if len(data) >= 7 && data[:7] == "base64," {
			data = data[7:]
		}
		return &geminiInlineData{MimeType: mime, Data: data}
	}
	return &geminiInlineData{}
}

// isHTTPURLPrefix 复刻 isUrl：url.Parse 后 scheme 为 http / https（Parse 会把 scheme 转小写）。
func isHTTPURLPrefix(b []byte) bool {
	l := make([]byte, 0, 8)
	for i := 0; i < len(b) && i < 8; i++ {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		l = append(l, c)
	}
	return bytes.HasPrefix(l, []byte("http://")) || bytes.HasPrefix(l, []byte("https://"))
}

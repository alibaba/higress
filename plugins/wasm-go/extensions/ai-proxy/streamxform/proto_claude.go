package streamxform

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// OpenAI → Claude 的流式转换协议。
//
// 逐行对照官方 claude.go buildClaudeTextGenRequest 推导。原则：
//   - 长的东西（messages[].content 文本、图片 base64）流式直通，只在两端加壳；
//   - 小的东西（model / role / tools / tool_calls / thinking 配置）Capture 后用与官方
//     相同的 struct 走一遍 encoding/json，字节级复刻官方的 omitempty 与字段形状；
//   - 依赖后续字段的决定（content 等 role、assistant content 等 tool_calls、part 等 type）
//     用有界 Defer，超上限 Bail——不猜。
//   - 官方 struct 里没有的字段官方会静默丢弃，这里同样 Skip；这是与官方一致的正确行为。
//
// 已知与官方的差异（都是"更宽松"，不会产出语义不同的合法请求）：
//   - 被 Skip 的字段若类型不合法，官方 Unmarshal 会整体失败，这里不会；
//   - 非法的 image data URL 官方会跳过该 part 并记日志，这里 Bail。

type ClaudeOptions struct {
	// MapModel 复刻官方 mapModel：model 为空或映射结果为空时返回错误。nil = 原样。
	MapModel func(model string) (string, error)
	// ClaudeCodeMode 对应 provider 配置 claudeCodeMode：system 走数组 + cache_control。
	ClaudeCodeMode bool
}

const (
	claudeDefaultMaxTokens        = 4096
	claudeMinThinkingBudgetTokens = 1024
	claudeCodeSystemPrompt        = "You are Claude Code, Anthropic's official CLI for Claude."

	// roleWaitCap：content 先于 role 到达时最多暂存多少；超过 Bail，不猜 role。
	roleWaitCap = 64 << 10
	// assistantWaitCap：assistant 的 content 要等 tool_calls 才能定形状。
	assistantWaitCap = 1 << 20
	// partWaitCap：多模态 part 里 type 迟到时暂存 text / image_url。
	partWaitCap = 8 << 20
	smallCap    = 64 << 10
	toolsCap    = 4 << 20
	// systemCap：system 内容必须整体搬到顶层。官方路径的请求体上限是 100MB（ai-proxy defaultMaxBodyBytes），
	// 这里对齐它——不能因为改成流式反而让单个字段可以无限增长。
	systemCap    = 100 << 20
	urlPrefixWin = 512
)

// ---- 与官方逐字段对齐的小结构（只用于 Capture 后的改写）----

type oaiTool struct {
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}
type oaiFunction struct {
	Description string                 `json:"description,omitempty"`
	Name        string                 `json:"name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}
type oaiToolCall struct {
	Index    int             `json:"index"`
	Id       string          `json:"id,omitempty"`
	Type     string          `json:"type"`
	Function oaiFunctionCall `json:"function"`
}
type oaiFunctionCall struct {
	Id        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments"`
}
type oaiToolChoice struct {
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}
type claudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}
type claudeToolChoice struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}
type claudeThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
	Display      string `json:"display,omitempty"`
}
type claudeOutputConfig struct {
	Effort string          `json:"effort,omitempty"`
	Format json.RawMessage `json:"format,omitempty"`
}
type claudeToolUse struct {
	Type  string                  `json:"type"`
	Id    string                  `json:"id,omitempty"`
	Name  string                  `json:"name,omitempty"`
	Input *map[string]interface{} `json:"input,omitempty"`
}
type claudeToolResult struct {
	Type      string `json:"type"`
	ToolUseId string `json:"tool_use_id,omitempty"`
	Content   string `json:"content"`
}

type claudePart struct {
	typ     string
	typSeen bool
	dead    bool // 官方会跳过这个 part
	urlSeen bool
	file    []byte
}

type claudeMsg struct {
	role           string
	roleSeen       bool
	roleWritten    bool
	contentSeen    bool
	contentWritten bool
	toolCallId     string
	toolCalls      []byte // 原始字节；nil = 未见
	toolText       string // tool role 的文本内容
	finalizing     bool
	part           claudePart
}

type claudeProto struct {
	opt ClaudeOptions

	model         string
	modelSeen     bool
	maxTok        int
	maxCompletion int
	stream        bool
	streamSeen    bool
	temp, topP    []byte
	stop          []byte
	stopN         int
	reasonEffort  string
	reasonMax     int
	thinkingRaw   []byte
	outputRaw     []byte
	toolsRaw      []byte
	toolChoiceRaw []byte
	parallelTC    *bool

	messagesSeen   bool
	inputMsgs      int
	msgCount       int
	msgLevel       int
	openToolResult bool
	sysSeen        bool
	sys            string // 解码后的文本（content 是数组时）
	sysRaw         []byte // content 是字符串时的原始 JSON 字面量，原样写出
	m              claudeMsg
}

// NewClaude 构造 OpenAI → Claude 转换器。
func NewClaude(opt ClaudeOptions) *Transformer {
	if opt.MapModel == nil {
		// 官方 mapModel 的最小语义：model 缺失即失败
		opt.MapModel = func(m string) (string, error) {
			if m == "" {
				return "", errors.New("missing model in request")
			}
			return m, nil
		}
	}
	p := &claudeProto{opt: opt}
	t := NewTransformer(p)
	t.DupKeyBail = true // 官方 struct 解析是"后者覆盖前者"，流式无法复刻，回落
	return t
}

// New 保留旧接口：默认选项的 Claude 转换器。
func New() *Transformer { return NewClaude(ClaudeOptions{}) }

func (p *claudeProto) Prelude() Prelude {
	return Prelude{Model: p.model, ModelSeen: p.modelSeen, Stream: p.stream, StreamSeen: p.streamSeen}
}

// ---- 派发 ----

func (p *claudeProto) OnKey(t *Transformer) Action {
	switch t.Depth() {
	case 1:
		return p.topKey(t)
	case 3:
		return p.msgKey(t)
	case 5:
		return p.partKey(t)
	case 6:
		return p.imageKey(t)
	}
	return Bail("意外的路径: " + t.PathString())
}

func (p *claudeProto) OnElem(t *Transformer) Action {
	switch t.Depth() {
	case 2, 4: // messages[i] / messages[i].content[j]
		return Probe()
	}
	return Bail("意外的数组: " + t.PathString())
}

func (p *claudeProto) OnStart(t *Transformer, kind ValueKind) Action {
	w := t.W()
	switch t.Depth() {
	case 1: // messages
		if kind != KindArray {
			return Bail("messages 不是数组")
		}
		p.msgLevel = w.Level() + 1
		return Enter().Lazy() // 全是 system 时官方输出 null，由 Tail 补
	case 2: // messages[i]
		if kind != KindObject {
			return Bail("message 不是对象")
		}
		p.m = claudeMsg{}
		p.inputMsgs++
		return Enter().Lazy() // system / 被合并的 tool 消息不产生元素
	case 3: // messages[i].content（role 已知且为普通消息）
		switch kind {
		case KindString:
			p.writeRole(t)
			p.m.contentWritten = true
			return Pass()
		case KindArray:
			p.writeRole(t)
			p.m.contentWritten = true
			return Enter()
		}
		// 对象 / 标量 / null：官方 ParseContent 得到空 → "content":[]，在 OnLeave 补
		return Skip()
	case 4: // messages[i].content[j]
		if kind != KindObject {
			return Skip() // 官方：非 map 的元素直接跳过
		}
		p.m.part = claudePart{}
		return Enter().Lazy() // 官方会跳过的 part 不留痕迹
	case 5:
		pt := &p.m.part
		switch t.Last() {
		case "text":
			if kind != KindString {
				pt.dead = true // 官方：text 不是字符串则整个 part 跳过
				return Skip()
			}
			w.Key("type")
			w.RawString(`"text"`)
			return Prefix(1) // 官方 Text 带 omitempty：空串时不输出 text 键，得先看一眼是不是空

		case "image_url":
			if kind != KindObject {
				pt.dead = true
				return Skip()
			}
			return Enter().Flat()
		}
	}
	return Bail("意外的 Probe: " + t.PathString())
}

func (p *claudeProto) topKey(t *Transformer) Action {
	switch t.Last() {
	case "model":
		return Capture(4 << 10)
	case "messages":
		return Probe()
	case "max_tokens", "max_completion_tokens", "reasoning_max_tokens", "temperature", "top_p":
		return Capture(64)
	case "stream", "parallel_tool_calls":
		return Capture(16)
	case "stop":
		return Capture(smallCap)
	case "reasoning_effort":
		return Capture(256)
	case "claude_thinking":
		return Capture(4 << 10)
	case "claude_output_config":
		return Capture(smallCap)
	case "tools":
		return Capture(toolsCap)
	case "tool_choice":
		return Capture(4 << 10)
	}
	// Claude 无对应字段，或官方 chatCompletionRequest 未定义：官方同样丢弃
	return Skip()
}

func (p *claudeProto) msgKey(t *Transformer) Action {
	m := &p.m
	switch t.Last() {
	case "role":
		return Capture(256)
	case "content":
		if !m.roleSeen {
			return Defer(roleWaitCap)
		}
		m.contentSeen = true
		switch m.role {
		case "system", "developer":
			return Capture(systemCap) // 必须整体搬到顶层 system；上限对齐官方的请求体上限
		case "tool":
			return Capture(assistantWaitCap)
		case "assistant":
			if !m.finalizing {
				return Defer(assistantWaitCap) // 形状取决于是否有 tool_calls
			}
		}
		return Probe()
	case "tool_calls":
		return Capture(assistantWaitCap)
	case "tool_call_id":
		return Capture(4 << 10)
	case "claude_content_blocks":
		return Bail("claude_content_blocks 需要 struct 往返，流式未复刻")
	}
	return Skip() // name / audio / refusal / reasoning* / function_call …：官方不读
}

func (p *claudeProto) partKey(t *Transformer) Action {
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
	case "file":
		if !pt.typSeen {
			return Defer(smallCap)
		}
		if pt.typ != "file" {
			return Skip()
		}
		return Capture(smallCap)
	}
	return Skip() // cache_control / input_audio / 未知：官方不读
}

func (p *claudeProto) imageKey(t *Transformer) Action {
	if t.Last() == "url" {
		p.m.part.urlSeen = true
		return Prefix(urlPrefixWin)
	}
	return Skip() // detail 等
}

// ---- 值到齐 ----

func (p *claudeProto) OnValue(t *Transformer, raw []byte) {
	switch t.Depth() {
	case 1:
		p.topValue(t, raw)
	case 3:
		p.msgValue(t, raw)
	case 5:
		p.partValue(t, raw)
	}
}

func (p *claudeProto) topValue(t *Transformer, raw []byte) {
	isNull := string(raw) == "null"
	switch t.Last() {
	case "model":
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("model 不是字符串")
			return
		}
		p.model, p.modelSeen = s, true
	case "max_tokens", "max_completion_tokens", "reasoning_max_tokens":
		if isNull {
			return
		}
		if !isIntLiteral(raw) {
			t.Bail(t.Last() + " 不是整数")
			return
		}
		n := atoi(raw)
		switch t.Last() {
		case "max_tokens":
			p.maxTok = n
		case "max_completion_tokens":
			p.maxCompletion = n
		default:
			p.reasonMax = n
		}
	case "stream":
		switch string(raw) {
		case "true":
			p.stream = true
		case "false", "null":
			p.stream = false
		default:
			t.Bail("stream 不是布尔")
			return
		}
		p.streamSeen = !isNull
	case "parallel_tool_calls":
		switch string(raw) {
		case "true":
			v := true
			p.parallelTC = &v
		case "false":
			v := false
			p.parallelTC = &v
		case "null":
		default:
			t.Bail("parallel_tool_calls 不是布尔")
		}
	case "temperature", "top_p":
		if isNull {
			return
		}
		if !isNumLiteral(raw) {
			t.Bail(t.Last() + " 不是数字")
			return
		}
		cp := append([]byte(nil), raw...)
		if t.Last() == "temperature" {
			p.temp = cp
		} else {
			p.topP = cp
		}
	case "stop":
		if isNull {
			return
		}
		var ss []string
		if err := json.Unmarshal(raw, &ss); err != nil {
			t.Bail("stop 不是字符串数组")
			return
		}
		p.stop = append([]byte(nil), raw...)
		p.stopN = len(ss)
	case "reasoning_effort":
		if isNull {
			return
		}
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("reasoning_effort 不是字符串")
			return
		}
		p.reasonEffort = s
	case "claude_thinking":
		if !isNull {
			p.thinkingRaw = append([]byte(nil), raw...)
		}
	case "claude_output_config":
		if !isNull {
			p.outputRaw = append([]byte(nil), raw...)
		}
	case "tools":
		if !isNull {
			p.toolsRaw = append([]byte(nil), raw...)
		}
	case "tool_choice":
		if !isNull {
			p.toolChoiceRaw = append([]byte(nil), raw...)
		}
	}
}

func (p *claudeProto) msgValue(t *Transformer, raw []byte) {
	m := &p.m
	switch t.Last() {
	case "role":
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("role 不是字符串")
			return
		}
		m.role, m.roleSeen = s, true
		// 官方：system 消息 continue 掉，不影响"上一条输出消息"的判定，
		// 所以 tool_result 的合并可以跨过 system 消息。
		if p.openToolResult && s != "tool" && s != "system" && s != "developer" {
			p.closeToolResult(t)
		}
		if s != "assistant" && len(t.Deferred()) > 0 {
			t.Release() // content 先到了：现在知道 role，回放
		}
	case "content":
		// 只有 system / developer / tool 的 content 会 Capture 到这里
		switch m.role {
		case "system", "developer":
			p.sysSeen = true
			if len(raw) > 0 && raw[0] == '"' {
				p.sysRaw = append([]byte(nil), raw...) // 字符串：不解码，原样写出
				p.sys = ""
			} else {
				p.sysRaw = nil
				p.sys = stringContent(t, raw)
			}
		case "tool":
			m.toolText = toolResultText(t, raw)
		}
	case "tool_calls":
		if string(raw) != "null" {
			m.toolCalls = append([]byte(nil), raw...)
		}
	case "tool_call_id":
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("tool_call_id 不是字符串")
			return
		}
		m.toolCallId = s
	}
}

func (p *claudeProto) partValue(t *Transformer, raw []byte) {
	pt := &p.m.part
	switch t.Last() {
	case "type":
		pt.typSeen = true
		s, ok := jsonUnquote(raw)
		if !ok {
			pt.dead = true // 官方 switch contentMap["type"] 匹配不到
			t.DropDeferred()
			return
		}
		pt.typ = s
		switch s {
		case "text", "image_url", "file":
			if len(t.Deferred()) > 0 {
				t.Release()
			}
		default:
			pt.dead = true // input_audio 官方明确不支持；其余匹配不到
			t.DropDeferred()
		}
	case "file":
		pt.file = append([]byte(nil), raw...)
	}
}

// OnPrefix：image_url.url 的前缀窗口。复刻官方对 data: URL 的拆分。
func (p *claudeProto) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	w := t.W()
	if t.Depth() == 5 && t.Last() == "text" {
		if complete && len(raw) == 0 {
			return Skip(), 0 // {"type":"text"}，与官方 omitempty 一致
		}
		w.Key("text")
		return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
	}
	dec, off := unescapePrefix(raw)
	if !bytes.HasPrefix(dec, []byte("data:")) {
		w.Key("type")
		w.RawString(`"image"`)
		w.Key("source")
		if complete && len(dec) == 0 {
			w.RawString(`{"type":"url"}`) // Url omitempty
			return Skip(), 0
		}
		return Pass().Wrap([]byte(`{"type":"url","url":"`), []byte(`"}`)), 0
	}
	semi := bytes.IndexByte(dec, ';')
	if semi < 0 {
		if complete {
			return Bail("image url 格式非法，官方会跳过该 part"), 0
		}
		return Bail("data URL 头超出前缀窗口"), 0
	}
	media := string(dec[5:semi])
	rest := dec[semi+1:]
	resumeDec := semi + 1
	if !complete && len(rest) < len("base64,") {
		return Bail("data URL 头在窗口边界被截断"), 0
	}
	if bytes.HasPrefix(rest, []byte("base64,")) {
		resumeDec += len("base64,")
	}
	resume := off[resumeDec]
	dataEmpty := complete && resume >= len(raw)
	if !complete && resume >= len(raw) {
		return Bail("data URL 数据段在窗口边界被截断"), 0
	}
	w.Key("type")
	w.RawString(`"image"`)
	w.Key("source")
	var pre []byte
	pre = append(pre, `{"type":"base64"`...)
	if media != "" {
		pre = append(pre, `,"media_type":`...)
		pre = appendJSONString(pre, media)
	}
	if dataEmpty {
		w.Raw(pre)
		w.Byte('}')
		return Skip(), 0
	}
	pre = append(pre, `,"data":"`...)
	return Pass().Wrap(pre, []byte(`"}`)), resume
}

// ---- 容器闭合 ----

func (p *claudeProto) OnLeave(t *Transformer) {
	w := t.W()
	switch t.Depth() {
	case 1: // messages
		if p.openToolResult {
			p.closeToolResult(t)
		}
		p.messagesSeen = true
		if p.inputMsgs == 0 {
			t.Bail("no message found in the request body")
		}
	case 2: // messages[i]
		p.finishMessage(t)
	case 3: // messages[i].content 数组：官方总是物化 []
		w.Open()
	case 4: // part
		pt := &p.m.part
		if pt.dead || !pt.typSeen {
			t.DropDeferred()
			return
		}
		if pt.typ == "file" && pt.file != nil {
			var obj map[string]interface{}
			if err := json.Unmarshal(pt.file, &obj); err != nil {
				t.Bail("file 不是对象")
				return
			}
			id, ok := obj["file_id"].(string)
			if !ok {
				t.Bail("file.file_id 缺失，官方会 panic")
				return
			}
			w.Key("type")
			w.RawString(`"file"`)
			w.Key("source")
			w.RawString(`{"type":"url"`)
			if id != "" {
				w.RawString(`,"file_id":`)
				w.JSONString(id)
			}
			w.Byte('}')
		}
	case 5: // image_url
		if !p.m.part.urlSeen {
			t.Bail("image_url.url 缺失，官方会 panic")
		}
	}
}

func (p *claudeProto) writeRole(t *Transformer) {
	if p.m.roleWritten {
		return
	}
	p.m.roleWritten = true
	p.msgCount++
	w := t.W()
	w.Key("role")
	w.JSONString(p.m.role)
}

func (p *claudeProto) closeToolResult(t *Transformer) {
	t.W().RawString("]}")
	p.openToolResult = false
}

// finishMessage 在一条 message 闭合时定稿。
func (p *claudeProto) finishMessage(t *Transformer) {
	m := &p.m
	w := t.W()
	if !m.roleSeen {
		m.role, m.roleSeen = "", true // 官方 Role 零值
		if p.openToolResult {
			p.closeToolResult(t)
		}
	}
	m.finalizing = true
	switch m.role {
	case "system", "developer":
		if len(t.Deferred()) > 0 {
			t.ReleaseNow()
		}
		if !m.contentSeen {
			p.sysSeen, p.sys, p.sysRaw = true, "", nil
		}
		return
	case "tool":
		if len(t.Deferred()) > 0 {
			t.ReleaseNow()
			if t.dead {
				return
			}
		}
		blk, _ := json.Marshal(claudeToolResult{Type: "tool_result", ToolUseId: m.toolCallId, Content: m.toolText})
		if p.openToolResult {
			w.Byte(',')
			w.Raw(blk)
			return
		}
		if !w.ElemAt(p.msgLevel) {
			t.Bail("tool message 输出位置异常")
			return
		}
		w.RawString(`{"role":"user","content":[`)
		w.Raw(blk)
		p.openToolResult = true
		p.msgCount++
		return
	case "assistant":
		if m.toolCalls != nil {
			var tcs []oaiToolCall
			if err := json.Unmarshal(m.toolCalls, &tcs); err != nil {
				t.Bail("tool_calls 解析失败: " + err.Error())
				return
			}
			if len(tcs) > 0 {
				p.writeAssistantWithTools(t, tcs)
				return
			}
		}
	}
	// 普通消息（user / assistant 无 tool_calls / 其他 role）
	if len(t.Deferred()) > 0 {
		t.ReleaseNow()
		if t.dead {
			return
		}
	}
	p.writeRole(t)
	if !m.contentWritten {
		w.Key("content")
		w.RawString("[]")
	}
}

func (p *claudeProto) writeAssistantWithTools(t *Transformer, tcs []oaiToolCall) {
	w := t.W()
	p.writeRole(t)
	w.Key("content")
	w.Byte('[')
	first := true
	for _, kv := range t.Deferred() {
		if kv.Key != "content" {
			continue
		}
		// 官方：IsStringContent && StringContent != ""
		if len(kv.Raw) > 2 && kv.Raw[0] == '"' {
			w.RawString(`{"type":"text","text":`)
			w.Raw(kv.Raw)
			w.Byte('}')
			first = false
		}
	}
	t.DropDeferred()
	for _, tc := range tcs {
		var input map[string]interface{}
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
				input = make(map[string]interface{})
			}
		} else {
			input = make(map[string]interface{})
		}
		blk, _ := json.Marshal(claudeToolUse{Type: "tool_use", Id: tc.Id, Name: tc.Function.Name, Input: &input})
		if !first {
			w.Byte(',')
		}
		w.Raw(blk)
		first = false
	}
	w.Byte(']')
}

// ---- 收尾 ----

func (p *claudeProto) Tail(t *Transformer) {
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
	if mapped != "" {
		w.Key("model")
		w.JSONString(mapped)
	}
	if p.msgCount == 0 {
		w.Key("messages")
		w.RawString("null") // 官方 nil slice
	}
	// system
	if p.opt.ClaudeCodeMode {
		w.Key("system")
		w.RawString(`[{"type":"text"`)
		switch {
		case !p.sysSeen:
			w.RawString(`,"text":`)
			w.JSONString(claudeCodeSystemPrompt)
		case p.sysRaw != nil:
			if string(p.sysRaw) != `""` { // Text omitempty
				w.RawString(`,"text":`)
				w.Raw(p.sysRaw)
			}
		case p.sys != "":
			w.RawString(`,"text":`)
			w.JSONString(p.sys)
		}
		w.RawString(`,"cache_control":{"type":"ephemeral"}}]`)
	} else if p.sysSeen {
		w.Key("system")
		if p.sysRaw != nil {
			w.Raw(p.sysRaw)
		} else {
			w.JSONString(p.sys)
		}
	}
	maxTokens := p.maxTok
	if p.maxCompletion > 0 {
		maxTokens = p.maxCompletion
	}
	if maxTokens == 0 {
		maxTokens = claudeDefaultMaxTokens
	}
	w.Key("max_tokens")
	w.Int(maxTokens)
	if p.stopN > 0 {
		w.Key("stop_sequences")
		w.Raw(p.stop)
	}
	if p.stream {
		w.Key("stream")
		w.RawString("true")
	}
	if p.temp != nil && !isZeroNum(p.temp) {
		w.Key("temperature")
		w.Raw(p.temp)
	}
	if p.topP != nil && !isZeroNum(p.topP) {
		w.Key("top_p")
		w.Raw(p.topP)
	}
	// thinking
	var thinking *claudeThinkingConfig
	if p.thinkingRaw != nil {
		var cfg claudeThinkingConfig
		if err := json.Unmarshal(p.thinkingRaw, &cfg); err != nil {
			t.Bail("claude_thinking 解析失败")
			return
		}
		thinking = &cfg
	}
	if thinking == nil && (p.reasonEffort != "" || p.reasonMax > 0) {
		var budget int
		if p.reasonMax > 0 {
			budget = p.reasonMax
		} else {
			switch p.reasonEffort {
			case "low":
				budget = 1024
			case "medium":
				budget = 8192
			case "high":
				budget = 16384
			default:
				budget = 8192
			}
		}
		if budget < claudeMinThinkingBudgetTokens {
			budget = claudeMinThinkingBudgetTokens
		}
		if budget >= maxTokens {
			budget = maxTokens - 1
		}
		if budget >= claudeMinThinkingBudgetTokens {
			thinking = &claudeThinkingConfig{Type: "enabled", BudgetTokens: budget}
		}
	}
	// tool_choice
	if p.toolChoiceRaw != nil {
		p.writeToolChoice(t, thinking)
		if t.dead {
			return
		}
	}
	// tools
	if p.toolsRaw != nil {
		var tools []oaiTool
		if err := json.Unmarshal(p.toolsRaw, &tools); err != nil {
			t.Bail("tools 解析失败: " + err.Error())
			return
		}
		if len(tools) > 0 {
			out := make([]claudeTool, 0, len(tools))
			for _, tl := range tools {
				out = append(out, claudeTool{Name: tl.Function.Name, Description: tl.Function.Description, InputSchema: tl.Function.Parameters})
			}
			b, _ := json.Marshal(out)
			w.Key("tools")
			w.Raw(b)
		}
	}
	if thinking != nil {
		b, _ := json.Marshal(thinking)
		w.Key("thinking")
		w.Raw(b)
	}
	if p.outputRaw != nil {
		var cfg claudeOutputConfig
		if err := json.Unmarshal(p.outputRaw, &cfg); err != nil {
			t.Bail("claude_output_config 解析失败")
			return
		}
		b, _ := json.Marshal(&cfg)
		w.Key("output_config")
		w.Raw(b)
	}
}

func (p *claudeProto) writeToolChoice(t *Transformer, thinking *claudeThinkingConfig) {
	w := t.W()
	var any interface{}
	if err := json.Unmarshal(p.toolChoiceRaw, &any); err != nil {
		t.Bail("tool_choice 解析失败")
		return
	}
	if any == nil {
		return
	}
	parallel := true
	if p.parallelTC != nil {
		parallel = *p.parallelTC
	}
	hasThinking := thinking != nil && thinking.Type != "" && thinking.Type != "disabled"

	var tcStr string
	var tcObj *oaiToolChoice
	switch v := any.(type) {
	case string:
		tcStr = v
	default:
		b, err := json.Marshal(any)
		if err == nil {
			var parsed oaiToolChoice
			if json.Unmarshal(b, &parsed) == nil {
				tcObj = &parsed
			}
		}
	}
	choiceType := tcStr
	if choiceType == "" && tcObj != nil {
		choiceType = tcObj.Type
	}
	var out *claudeToolChoice
	if !hasThinking && tcObj != nil && tcObj.Type == "function" && tcObj.Function.Name != "" {
		out = &claudeToolChoice{Name: tcObj.Function.Name, Type: "tool", DisableParallelToolUse: !parallel}
	} else if choiceType != "" {
		switch choiceType {
		case "required":
			choiceType = "any"
		case "function":
			choiceType = "auto"
		}
		if hasThinking && (choiceType == "any" || choiceType == "tool") {
			choiceType = "auto"
		}
		out = &claudeToolChoice{Type: choiceType}
		if choiceType != "none" {
			out.DisableParallelToolUse = !parallel
		}
	}
	if out != nil {
		b, _ := json.Marshal(out)
		w.Key("tool_choice")
		w.Raw(b)
	}
}

// ---- 官方 StringContent / ParseContent 的复刻（只用于已 Capture 的小值）----

// stringContent 复刻 chatMessage.StringContent：字符串原样；数组取 text 部分各加 "\n"；其余 ""。
func stringContent(t *Transformer, raw []byte) string {
	if len(raw) > 0 && raw[0] == '"' {
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("content 字符串非法")
		}
		return s
	}
	if len(raw) > 0 && raw[0] == '[' {
		var items []interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Bail("content 数组非法")
			return ""
		}
		var sb strings.Builder
		for _, it := range items {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "text" {
				if s, ok := m["text"].(string); ok {
					sb.WriteString(s)
					sb.WriteString("\n")
				}
			}
		}
		return sb.String()
	}
	return ""
}

// toolResultText 复刻 tool role 分支：字符串原样；否则 ParseContent 的 text 部分用 "\n" 连接。
func toolResultText(t *Transformer, raw []byte) string {
	if len(raw) > 0 && raw[0] == '"' {
		s, ok := jsonUnquote(raw)
		if !ok {
			t.Bail("content 字符串非法")
		}
		return s
	}
	if len(raw) > 0 && raw[0] == '[' {
		var items []interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Bail("content 数组非法")
			return ""
		}
		var parts []string
		for _, it := range items {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "text" {
				if s, ok := m["text"].(string); ok {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func atoi(b []byte) int {
	n, neg := 0, false
	for i, c := range b {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if i == 0 && c == '+' {
			continue
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		return -n
	}
	return n
}

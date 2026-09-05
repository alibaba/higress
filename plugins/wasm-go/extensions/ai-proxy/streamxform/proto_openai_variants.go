package streamxform

import (
	"encoding/json"
	"strconv"
)

// ---- Qwen 兼容模式（qwenEnableCompatible，默认开启）----
//
// 官方 TransformRequestBodyHeaders 兼容分支：model 存在才改写；映射结果非空、
// 某条 message 带非空 reasoning_content、且模型支持时，补 preserve_thinking:true。
// 不设置 Accept / isStreaming（官方这条分支不调 defaultTransformRequestBody）。

type QwenVariant struct {
	// SupportsPreserveThinking 复刻 qwenSupportsPreserveThinking。
	SupportsPreserveThinking func(model string) bool
	ptRaw                    []byte
	ptSeen                   bool
}

func (v *QwenVariant) TopKey(t *Transformer, key string) (Action, bool) {
	if key == "preserve_thinking" && !v.ptSeen {
		return Capture(64), true
	}
	return Action{}, false
}

func (v *QwenVariant) TopValue(t *Transformer, key string, raw []byte) {
	if key == "preserve_thinking" {
		v.ptSeen = true
		v.ptRaw = append([]byte(nil), raw...)
	}
}

func (v *QwenVariant) NeedReasoningScan() bool { return true }

func (v *QwenVariant) Tail(t *Transformer, st *OpenAIState) {
	w := t.W()
	if st.ModelSeen && st.Mapped != "" && st.ReasoningSeen && v.SupportsPreserveThinking != nil && v.SupportsPreserveThinking(st.Mapped) {
		w.Key("preserve_thinking")
		w.RawString("true")
		return
	}
	if v.ptSeen {
		w.Key("preserve_thinking")
		w.Raw(v.ptRaw)
	}
}

// ---- 智谱（chat completion）----
//
// 官方：reasoning_effort 非空 → thinking 整体替换为 {"type":"enabled"} 并删掉 reasoning_effort；
// 任一 message 带非空 reasoning_content → thinking.clear_thinking = false。

type ZhipuVariant struct {
	effortRaw  []byte
	effortSeen bool
	thinkRaw   []byte
	thinkSeen  bool
}

func (v *ZhipuVariant) TopKey(t *Transformer, key string) (Action, bool) {
	switch key {
	case "reasoning_effort":
		if !v.effortSeen {
			return Capture(256), true
		}
	case "thinking":
		if !v.thinkSeen {
			return Capture(4 << 10), true
		}
	}
	return Action{}, false
}

func (v *ZhipuVariant) TopValue(t *Transformer, key string, raw []byte) {
	switch key {
	case "reasoning_effort":
		v.effortSeen = true
		v.effortRaw = append([]byte(nil), raw...)
	case "thinking":
		v.thinkSeen = true
		v.thinkRaw = append([]byte(nil), raw...)
	}
}

func (v *ZhipuVariant) NeedReasoningScan() bool { return true }

func (v *ZhipuVariant) Tail(t *Transformer, st *OpenAIState) {
	w := t.W()
	var thinking []byte
	if v.thinkSeen {
		thinking = v.thinkRaw
	}
	if v.effortSeen && gjsonStringNonEmpty(v.effortRaw) {
		thinking = []byte(`{"type":"enabled"}`) // sjson 用 map 整体替换
	} else if v.effortSeen {
		w.Key("reasoning_effort")
		w.Raw(v.effortRaw)
	}
	if st.ReasoningSeen {
		var err error
		thinking, err = setObjectKey(thinking, "clear_thinking", []byte("false"))
		if err != nil {
			t.Bail("thinking 不是对象，sjson 的处理方式未复刻")
			return
		}
	}
	if thinking != nil {
		w.Key("thinking")
		w.Raw(thinking)
	}
}

// ---- OpenRouter（chat completion）----
//
// 官方：reasoning_max_tokens 存在且 Int() != 0 → 删 reasoning_effort、
// 置 reasoning.max_tokens、删 reasoning_max_tokens；否则原样。

type OpenRouterVariant struct {
	effortRaw  []byte
	effortSeen bool
	rmtRaw     []byte
	rmtSeen    bool
	reasonRaw  []byte
	reasonSeen bool
}

func (v *OpenRouterVariant) TopKey(t *Transformer, key string) (Action, bool) {
	switch key {
	case "reasoning_effort":
		if !v.effortSeen {
			return Capture(256), true
		}
	case "reasoning_max_tokens":
		if !v.rmtSeen {
			return Capture(64), true
		}
	case "reasoning":
		if !v.reasonSeen {
			return Capture(4 << 10), true
		}
	}
	return Action{}, false
}

func (v *OpenRouterVariant) TopValue(t *Transformer, key string, raw []byte) {
	cp := append([]byte(nil), raw...)
	switch key {
	case "reasoning_effort":
		v.effortSeen, v.effortRaw = true, cp
	case "reasoning_max_tokens":
		v.rmtSeen, v.rmtRaw = true, cp
	case "reasoning":
		v.reasonSeen, v.reasonRaw = true, cp
	}
}

func (v *OpenRouterVariant) NeedReasoningScan() bool { return false }

func (v *OpenRouterVariant) Tail(t *Transformer, st *OpenAIState) {
	w := t.W()
	n := int64(0)
	if v.rmtSeen {
		n = gjsonInt(v.rmtRaw)
	}
	if !v.rmtSeen || n == 0 {
		if v.effortSeen {
			w.Key("reasoning_effort")
			w.Raw(v.effortRaw)
		}
		if v.rmtSeen {
			w.Key("reasoning_max_tokens")
			w.Raw(v.rmtRaw)
		}
		if v.reasonSeen {
			w.Key("reasoning")
			w.Raw(v.reasonRaw)
		}
		return
	}
	var reasoning []byte
	if v.reasonSeen {
		reasoning = v.reasonRaw
	}
	reasoning, err := setObjectKey(reasoning, "max_tokens", []byte(strconv.FormatInt(n, 10)))
	if err != nil {
		t.Bail("reasoning 不是对象，sjson 的处理方式未复刻")
		return
	}
	w.Key("reasoning")
	w.Raw(reasoning)
}

// setObjectKey 复刻 sjson 对 "obj.key" 的设置：obj 缺失则新建；存在则替换/追加该 key。
func setObjectKey(obj []byte, key string, val []byte) ([]byte, error) {
	if obj == nil || string(obj) == "null" {
		return []byte(`{"` + key + `":` + string(val) + `}`), nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(obj, &m); err != nil || m == nil {
		return nil, errNotObject
	}
	m[key] = json.RawMessage(val)
	return json.Marshal(m)
}

type notObjectError struct{}

func (notObjectError) Error() string { return "not an object" }

var errNotObject error = notObjectError{}

// gjsonInt 复刻 gjson.Result.Int() 的主要分支：数字截断为整数；字符串按整数解析；true 为 1。
func gjsonInt(raw []byte) int64 {
	if len(raw) == 0 {
		return 0
	}
	switch raw[0] {
	case 't':
		return 1
	case '"':
		s, ok := jsonUnquote(raw)
		if !ok {
			return 0
		}
		return gjsonParseInt(s) // gjson 对字符串只认纯数字（可带负号），其余为 0
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if f, err := strconv.ParseFloat(string(raw), 64); err == nil {
			return int64(f) // safeInt / 标准转换：截断
		}
	}
	return 0
}

func gjsonParseInt(s string) int64 {
	var n int64
	i := 0
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		i++
	}
	if i == len(s) {
		return 0
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0
		}
		n = n*10 + int64(s[i]-'0')
	}
	if neg {
		return -n
	}
	return n
}

package streamxform

// 可复用的子 hook：OpenAI `tools` 数组的逐元素流式映射。
//
// 官方 Claude / Gemini 都把 `tools[i].function` 解成同一个 function struct
// （description omitempty、name 必有、parameters 为空 map 或 nil 时省略）再各自包一层。
// 这里把"逐元素直通、parameters 内部原样、omitempty 语义"做成协议可以嵌入的小部件：
// 协议只决定外层怎么包、parameters 在输出侧叫什么。
//
// 用法：协议在 tools 相关路径上把回调转给它；约定 tools 数组本身位于路径深度 1
// （即 t.Key(0) == "tools"），元素在深度 2，function 在深度 3，function 的字段在深度 4，
// parameters 内部 ≥ 5。元素对象由 hook 自己 Enter；tools 数组本身由协议决定怎么进
// （Claude 直接 Enter().Lazy()，Gemini 先 Push 三层再 Enter().Flat()）。
type ToolsHook struct {
	// ParamsKey 输出侧 parameters 的 key（Claude 是 input_schema，Gemini 是 parameters）。
	ParamsKey string

	elem struct {
		nameSeen bool
		fnSeen   bool
	}
}

// OnElem：tools[i]
func (h *ToolsHook) OnElem(t *Transformer) Action { return Probe() }

// OnElemStart：tools[i] 的值类型判定
func (h *ToolsHook) OnElemStart(t *Transformer, kind ValueKind) Action {
	if kind == KindNull { // 官方解成零值 struct，仍输出一个元素
		w := t.W()
		w.Elem()
		w.RawString(`{"name":""}`)
		return Skip()
	}
	if kind != KindObject {
		return Bail("tools 元素不是对象，官方 struct 解析失败")
	}
	h.elem.nameSeen, h.elem.fnSeen = false, false
	return Enter() // 官方总会输出这个元素（哪怕 function 缺失，也有 "name":""）
}

// OnKey：tools 下的 key（深度 3 起）
func (h *ToolsHook) OnKey(t *Transformer) Action {
	switch t.Depth() {
	case 3: // tools[i].K：官方 tool struct 只读 function（type 不进输出）
		if t.Last() == "function" {
			return Probe()
		}
		return Skip()
	case 4: // tools[i].function.K
		switch t.Last() {
		case "name", "description", "parameters":
			return Probe()
		}
		return Skip()
	}
	return Pass() // parameters 内部：原样
}

// OnKeyStart：tools 下的值类型判定
func (h *ToolsHook) OnKeyStart(t *Transformer, kind ValueKind) Action {
	switch t.Depth() {
	case 3: // function
		switch kind {
		case KindObject:
			h.elem.fnSeen = true
			return Enter().Flat()
		case KindNull: // struct 零值，等价于缺失
			return Skip()
		}
		return Bail("tools[].function 不是对象，官方 struct 解析失败")
	case 4:
		switch t.Last() {
		case "name":
			if kind != KindString {
				return Bail("tools[].function.name 不是字符串，官方 struct 解析失败")
			}
			h.elem.nameSeen = true
			return Pass()
		case "description":
			if kind != KindString {
				return Bail("tools[].function.description 不是字符串，官方 struct 解析失败")
			}
			return Prefix(1) // omitempty：空串不输出
		case "parameters":
			switch kind {
			case KindObject:
				return Enter().As(h.ParamsKey).Lazy() // 空对象：omitempty 省略
			case KindNull:
				return Skip()
			}
			return Bail("tools[].function.parameters 不是对象，官方 struct 解析失败")
		}
	}
	return Pass()
}

// OnPrefix：description 的空串判定
func (h *ToolsHook) OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int) {
	if complete && len(raw) == 0 {
		return Skip(), 0
	}
	t.W().KeyRaw(t.KeyRaw())
	return Pass().Wrap([]byte(`"`), []byte(`"`)), 0
}

// OnLeave：tools[i] 闭合时补官方 struct 里没有 omitempty 的 name
func (h *ToolsHook) OnLeave(t *Transformer) {
	if t.Depth() == 2 && (!h.elem.fnSeen || !h.elem.nameSeen) {
		w := t.W()
		w.Key("name")
		w.RawString(`""`)
	}
}

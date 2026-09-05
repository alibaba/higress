package streamxform

// Writer 是输出器。核心是"惰性层"：
// 进入一个容器时只登记，不写开括号；第一次在里面写东西时才把它（和所有未打开的祖先）
// 打开。到闭合时若从没打开过，就什么都不写——被丢弃的元素不留痕迹。
//
// 逗号由 Key/Elem 自动处理，协议代码不用关心输出顺序——这是"字段无序"在实现层的落点。
type Writer struct {
	buf    []byte
	frames []wframe
}

type wframe struct {
	key    string // 打开时先写的 key；"" = 数组元素或根
	raw    []byte // 非空时打开层用它代替 "key":（保留原始空白）/ 元素前空白
	isArr  bool
	opened bool
	n      int // 已写子项数
}

// Level 当前层号（根 = 0）。
func (w *Writer) Level() int { return len(w.frames) - 1 }

func (w *Writer) push(key string, raw []byte, isArr bool) {
	w.frames = append(w.frames, wframe{key: key, raw: raw, isArr: isArr})
}

// pop 闭合当前层；从未打开则不写任何东西。closeWs 是原文里闭合括号前的空白。
func (w *Writer) pop(closeWs []byte) {
	l := len(w.frames) - 1
	if w.frames[l].opened {
		w.buf = append(w.buf, closeWs...)
		if w.frames[l].isArr {
			w.buf = append(w.buf, ']')
		} else {
			w.buf = append(w.buf, '}')
		}
	}
	w.frames = w.frames[:l]
}

func (w *Writer) ensureOpen(level int) {
	f := &w.frames[level]
	if f.opened {
		return
	}
	if level > 0 {
		w.ensureOpen(level - 1)
		w.sep(level - 1)
		switch {
		case len(f.raw) > 0:
			w.buf = append(w.buf, f.raw...)
		case f.key != "":
			w.buf = append(w.buf, '"')
			w.buf = append(w.buf, f.key...)
			w.buf = append(w.buf, '"', ':')
		}
	}
	if f.isArr {
		w.buf = append(w.buf, '[')
	} else {
		w.buf = append(w.buf, '{')
	}
	f.opened = true
}

func (w *Writer) sep(level int) {
	f := &w.frames[level]
	if f.n > 0 {
		w.buf = append(w.buf, ',')
	}
	f.n++
}

// CanWriteAt 报告能否直接写到第 level 层：其上所有层都还没打开。
func (w *Writer) CanWriteAt(level int) bool {
	for l := level + 1; l < len(w.frames); l++ {
		if w.frames[l].opened {
			return false
		}
	}
	return true
}

// Opened 报告第 level 层是否已打开。
func (w *Writer) Opened(level int) bool { return w.frames[level].opened }

// KeyAt 在第 level 层写 "name":，自动处理逗号与祖先层的打开。
// 其上有已打开的层时返回 false 且不写。
func (w *Writer) KeyAt(level int, name string) bool {
	if !w.CanWriteAt(level) {
		return false
	}
	w.ensureOpen(level)
	w.sep(level)
	w.buf = append(w.buf, '"')
	w.buf = append(w.buf, name...)
	w.buf = append(w.buf, '"', ':')
	return true
}

// KeyRawAt 同 KeyAt，但 key 用原始字节（含空白与冒号）原样写出。
func (w *Writer) KeyRawAt(level int, raw []byte) bool {
	if !w.CanWriteAt(level) {
		return false
	}
	w.ensureOpen(level)
	w.sep(level)
	w.buf = append(w.buf, raw...)
	return true
}

// KeyRaw 作用于当前层。
func (w *Writer) KeyRaw(raw []byte) { w.KeyRawAt(w.Level(), raw) }

// ElemRawAt 同 ElemAt，并原样写出元素前的空白。
func (w *Writer) ElemRawAt(level int, ws []byte) bool {
	if !w.CanWriteAt(level) {
		return false
	}
	w.ensureOpen(level)
	w.sep(level)
	w.buf = append(w.buf, ws...)
	return true
}

// ElemAt 在第 level 层（数组）开始一个元素：只处理逗号与打开。
func (w *Writer) ElemAt(level int) bool {
	if !w.CanWriteAt(level) {
		return false
	}
	w.ensureOpen(level)
	w.sep(level)
	return true
}

// Key / Elem 作用于当前层。
func (w *Writer) Key(name string) { w.KeyAt(w.Level(), name) }
func (w *Writer) Elem()           { w.ElemAt(w.Level()) }

// Open 强制打开当前层（用于必须物化空容器的场合，如官方输出 "content":[]）。
func (w *Writer) Open() { w.ensureOpen(w.Level()) }

// Raw 原样追加字节。调用方负责它出现在语法上合法的位置。
func (w *Writer) Raw(b []byte)       { w.buf = append(w.buf, b...) }
func (w *Writer) RawString(s string) { w.buf = append(w.buf, s...) }
func (w *Writer) Byte(c byte)        { w.buf = append(w.buf, c) }

// JSONString 写一个带引号、已转义的 JSON 字符串。
func (w *Writer) JSONString(s string) { w.buf = appendJSONString(w.buf, s) }

// Int 写十进制整数。
func (w *Writer) Int(n int) { w.buf = appendInt(w.buf, n) }

// Len 当前未取走的输出字节数。
func (w *Writer) Len() int { return len(w.buf) }

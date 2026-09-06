// Package sjt 是流式跨协议 JSON 转换框架：
//
//	层 1 Scanner  —— 协议无关的字节级扫描器（本文件），把输入切成 key / 值 / 容器事件
//	层 2 Protocol —— 每协议一套手写 hooks（proto_*.go），对每个事件返回动作
//	层 3 Guard    —— 提交点 / 回落窗口（本文件的 committed / Bail 语义 + 集成层）
//
// 扫描器不建对象树。值要么原样流向输出（Pass），要么丢弃（Skip），
// 要么在协议明确要求时才进入有界缓冲（Capture / Defer / Prefix）。
// 内存与输入大小无关，只与协议要求缓冲的那几个小值有关。
package streamxform

import "encoding/json"

type scanState uint8

const (
	sIdle scanState = iota
	sInKey
	sInStr
	sInScalar
)

type frameKind uint8

const (
	fkObj frameKind = iota
	fkArr
)

type phase uint8

const (
	phKey   phase = iota // 对象：期待 key（或 }）
	phColon              // 对象：期待 :
	phValue              // 期待值（数组：或 ]）
	phComma              // 期待 , 或闭合
)

// frame 是一个 Enter 进来的容器（派发帧）。区域内部的容器不建帧，只计深度。
type frame struct {
	kind     frameKind
	ph       phase
	idx      int
	n        int  // 已完成的值个数（对象）
	flat     bool // 输出侧没有对应层
	lazy     bool // 闭合时不物化空容器
	seen     []string
	deferred []DeferredKV
}

// DeferredKV 是 Defer 暂存的一对 key/value 原始字节。
type DeferredKV struct {
	Key    string
	KeyRaw []byte // [空白]"key"[空白]:[空白]
	Raw    []byte
}

type seg struct {
	k string
	i int // -1 = key 段
}

type regionTarget uint8

const (
	rtNone regionTarget = iota
	rtOut
	rtSkip
	rtCapture
	rtDefer
	rtObserve
	rtPrefix
)

// CommitBytes 是提交点窗口：扫描这么多输入字节之前不下发任何输出。
// 越过之前判定不支持，调用方仍持有全部原始字节，可以干净回落。
const CommitBytes = 64 << 10

// Transformer 把一个 Protocol 接到扫描器上。Write 逐块喂入，Out 取出可下发的字节。
type Transformer struct {
	proto Protocol
	w     Writer

	// DupKeyBail：派发帧内出现重复 key 时判定不支持。
	// 目标协议用 struct 解析（后者覆盖前者）时应开启；字节透传类协议不需要。
	DupKeyBail bool

	st     scanState
	esc    bool
	hexN   uint8 // \u 转义还需读取的 hex 位数
	lit    litState
	depth  int
	frames []frame
	path   []seg
	keyBuf []byte
	keyEsc bool

	pend    Action
	pendSet bool

	// 原样保留派发帧里 key 周围的空白：kvRaw = [空白]"key"[空白]:[空白]，elemWs = 元素前空白。
	// 透传类协议靠它做到"没动的字节一个不改"，与 sjson 的原地修改效果一致。
	kvRaw       []byte
	wsRaw       []byte
	elemWs      []byte
	rootCloseWs []byte

	regOpen  bool
	regT     regionTarget
	regDepth int
	regInner bool
	regSuf   []byte
	regCap   int
	regKey   string
	capBuf   []byte

	wantRelease bool
	releaseAt   int // 请求回放时所在的派发帧号：只在该帧的安全点消费，进入子帧不会误消费

	scanned     int
	committed   bool
	unsupported bool
	reason      string
	dead        bool
	rootSeen    bool
	rootDone    bool
}

// NewTransformer 用指定协议构造转换器。
func NewTransformer(p Protocol) *Transformer {
	return &Transformer{proto: p}
}

// ---- 对外：Guard 语义 ----

// Committed 报告是否已越过提交点。越过之后再判定不支持，已发出的字节收不回来。
func (t *Transformer) Committed() bool { return t.committed }

// Unsupported 报告是否遇到了处理不了的输入。为 true 时输出不可用。
func (t *Transformer) Unsupported() (bool, string) { return t.unsupported, t.reason }

// Bail 由协议或框架调用：判定不支持，停止扫描。
func (t *Transformer) Bail(reason string) {
	if !t.unsupported {
		t.unsupported = true
		t.reason = reason
	}
	t.dead = true
}

// Out 取走可下发的字节。未越过提交点、或已判定不支持时返回空。
func (t *Transformer) Out() []byte {
	if t.unsupported || !t.committed || len(t.w.buf) == 0 {
		return nil
	}
	b := make([]byte, len(t.w.buf))
	copy(b, t.w.buf)
	t.w.buf = t.w.buf[:0]
	return b
}

// Write 喂入一块输入，可在任意字节边界切分。
func (t *Transformer) Write(p []byte) {
	if t.dead {
		return
	}
	t.scanned += len(p)
	t.scan(p)
	if !t.committed && !t.unsupported && t.scanned >= CommitBytes {
		t.committed = true
	}
}

// Finish 收尾：调用协议 Tail，闭合根对象。
// 若在此判定不支持，committed 保持原值——集成层据此决定回落还是失败。
func (t *Transformer) Finish() []byte {
	if t.dead {
		return nil
	}
	if t.st == sInScalar {
		t.scan([]byte{' '})
	}
	if !t.rootDone {
		t.Bail("输入不是完整的 JSON 对象")
		return nil
	}
	t.proto.Tail(t)
	if t.unsupported {
		return nil
	}
	t.w.ensureOpen(0)
	t.w.pop(t.rootCloseWs)
	t.committed = true
	return t.Out()
}

// ---- 对协议：路径与输出 ----

// Protocol 返回接入的协议（集成层用它取 Prelude）。
func (t *Transformer) Protocol() Protocol { return t.proto }

// W 输出器。
func (t *Transformer) W() *Writer { return &t.w }

// Depth 当前路径段数。
func (t *Transformer) Depth() int { return len(t.path) }

// Key 第 level 段的 key；该段是数组下标时返回 ""。
func (t *Transformer) Key(level int) string {
	if level < 0 || level >= len(t.path) {
		return ""
	}
	return t.path[level].k
}

// Idx 第 level 段的数组下标；该段是 key 时返回 -1。
func (t *Transformer) Idx(level int) int {
	if level < 0 || level >= len(t.path) {
		return -1
	}
	return t.path[level].i
}

// Last 最后一段的 key。
func (t *Transformer) Last() string { return t.Key(len(t.path) - 1) }

// PathString 调试用："messages[1].content"。
func (t *Transformer) PathString() string {
	var b []byte
	for i, s := range t.path {
		if s.i >= 0 {
			b = append(b, '[')
			b = appendInt(b, s.i)
			b = append(b, ']')
		} else {
			if i > 0 {
				b = append(b, '.')
			}
			b = append(b, s.k...)
		}
	}
	return string(b)
}

// KeyRaw 当前 key 的原始字节（含前导空白、引号、冒号及其周围空白）。协议想原样保留格式时用它。
func (t *Transformer) KeyRaw() []byte { return t.kvRaw }

// Release 请求回放当前派发帧里 Defer 的项。回放发生在当前回调返回后、同一帧的安全点
// （当前值结束或该帧闭合）；若回调返回 Enter 进入了子帧，回放推迟到回到本帧之后。
func (t *Transformer) Release() {
	t.wantRelease = true
	t.releaseAt = len(t.frames) - 1
}

// ReleaseNow 同步回放当前派发帧里 Defer 的项。只能在 OnLeave 里调用——
// 那时路径正指向容器本身，回放的 key 会正确地挂在它下面；在 OnValue 里要用 Release。
func (t *Transformer) ReleaseNow() { t.doRelease() }

// Deferred 查看当前派发帧里 Defer 的项。
func (t *Transformer) Deferred() []DeferredKV {
	if f := t.top(); f != nil {
		return f.deferred
	}
	return nil
}

// DropDeferred 丢弃当前派发帧里 Defer 的项。
func (t *Transformer) DropDeferred() {
	if f := t.top(); f != nil {
		f.deferred = nil
	}
}

// ---- 扫描器 ----

func (t *Transformer) top() *frame {
	if len(t.frames) == 0 {
		return nil
	}
	return &t.frames[len(t.frames)-1]
}

func (t *Transformer) scan(p []byte) {
	rs := -1
	if t.regOpen {
		rs = 0
	}
	i := 0
	for i < len(p) && !t.dead {
		c := p[i]
		switch t.st {
		case sInStr:
			if t.esc {
				t.esc = false
				switch escapeClass(c) {
				case 0:
					t.Bail("字符串里有非法转义")
					continue
				case 2:
					t.hexN = 4
				}
				i++
				continue
			}
			if t.hexN > 0 {
				if !isHexByte(c) {
					t.Bail("\\u 转义后不是 4 位十六进制")
					continue
				}
				t.hexN--
				i++
				continue
			}
			j := scanStringBody(p, i)
			if j == len(p) {
				i = j
				continue
			}
			i = j
			if p[i] < 0x20 {
				t.Bail("字符串里有未转义的控制字符")
				continue
			}
			if p[i] == '\\' {
				t.esc = true
				i++
				continue
			}
			// 未转义的引号：字符串结束
			t.st = sIdle
			if t.regOpen && t.depth == t.regDepth {
				end := i + 1
				if t.regInner {
					end = i
				}
				rs = t.flush(p, rs, end)
				t.endRegion()
				t.afterValue()
			}
			i++
		case sInKey:
			if t.esc {
				t.esc = false
				switch escapeClass(c) {
				case 0:
					t.Bail("key 里有非法转义")
					continue
				case 2:
					t.hexN = 4
				}
				t.keyBuf = append(t.keyBuf, c)
				t.kvRaw = append(t.kvRaw, c)
				i++
				continue
			}
			if t.hexN > 0 {
				if !isHexByte(c) {
					t.Bail("\\u 转义后不是 4 位十六进制")
					continue
				}
				t.hexN--
				t.keyBuf = append(t.keyBuf, c)
				t.kvRaw = append(t.kvRaw, c)
				i++
				continue
			}
			if c < 0x20 {
				t.Bail("key 里有未转义的控制字符")
				continue
			}
			if c == '\\' {
				t.esc = true
				t.keyEsc = true
				t.keyBuf = append(t.keyBuf, c)
				t.kvRaw = append(t.kvRaw, c)
				i++
				continue
			}
			if c == '"' {
				t.st = sIdle
				t.kvRaw = append(t.kvRaw, c)
				t.onKeyDone()
				i++
				continue
			}
			t.keyBuf = append(t.keyBuf, c)
			t.kvRaw = append(t.kvRaw, c)
			i++
		case sInScalar:
			if isScalarByte(c) {
				if t.lit.kind == KindNumber { // 内联的表驱动 DFA：数字是区域内最常见的标量
					t.lit.num = numStep(t.lit.num, c)
					if t.lit.num == nsBad {
						t.Bail("非法的标量字面量")
						continue
					}
				} else if !t.lit.step(c) {
					t.Bail("非法的标量字面量")
					continue
				}
				i++
				continue
			}
			if !t.lit.done() {
				t.Bail("不完整的标量字面量")
				continue
			}
			t.st = sIdle
			if t.regOpen && t.depth == t.regDepth {
				rs = t.flush(p, rs, i)
				t.endRegion()
				t.afterValue()
			}
			// 不消费 c，回到 sIdle 处理
		case sIdle:
			if jsonSpace[c] {
				if !t.regOpen && t.rootSeen && !t.rootDone {
					t.wsRaw = append(t.wsRaw, c)
				}
				i++
				continue
			}
			if t.regOpen {
				// 区域内部：只跟踪结构，不派发
				switch c {
				case '"':
					t.st = sInStr
					t.esc = false
				case '{', '[':
					t.depth++
				case '}', ']':
					t.depth--
					if t.depth == t.regDepth {
						rs = t.flush(p, rs, i+1)
						t.endRegion()
						t.afterValue()
					} else if t.depth < t.regDepth {
						t.Bail("JSON 结构不平衡")
					}
				case ',', ':':
				default:
					if !t.lit.start(c) {
						t.Bail("非法字符")
						continue
					}
					t.st = sInScalar
				}
				i++
				continue
			}
			if t.rootDone {
				t.Bail("根对象之后有多余内容")
				continue
			}
			f := t.top()
			if f == nil {
				// 根
				if c != '{' {
					t.Bail("根不是 JSON 对象")
					continue
				}
				t.rootSeen = true
				t.depth = 1
				t.frames = append(t.frames, frame{kind: fkObj, ph: phKey, idx: -1})
				t.w.push("", nil, false)
				t.wsRaw = t.wsRaw[:0]
				i++
				continue
			}
			switch c {
			case '}', ']':
				if (c == '}') != (f.kind == fkObj) {
					t.Bail("括号不匹配")
					continue
				}
				if f.kind == fkObj && (f.ph == phColon || f.ph == phValue) {
					t.Bail("key 后缺少值")
					continue
				}
				if f.kind == fkArr && f.ph == phValue && f.idx >= 0 {
					t.Bail("数组末尾多余逗号")
					continue
				}
				if f.kind == fkObj && f.ph == phKey && f.n > 0 {
					t.Bail("对象末尾多余逗号")
					continue
				}
				t.closeContainer()
				i++
			case ':':
				if f.kind != fkObj || f.ph != phColon {
					t.Bail("意外的冒号")
					continue
				}
				t.kvRaw = append(t.kvRaw, t.wsRaw...)
				t.kvRaw = append(t.kvRaw, ':')
				t.wsRaw = t.wsRaw[:0]
				f.ph = phValue
				i++
			case ',':
				if f.ph != phComma {
					t.Bail("意外的逗号")
					continue
				}
				t.wsRaw = t.wsRaw[:0] // 值与逗号之间的空白不保留
				if f.kind == fkObj {
					f.ph = phKey
				} else {
					f.ph = phValue
				}
				i++
			case '"':
				if f.kind == fkObj && f.ph == phKey {
					t.st = sInKey
					t.esc = false
					t.keyEsc = false
					t.keyBuf = t.keyBuf[:0]
					t.kvRaw = append(t.kvRaw[:0], t.wsRaw...)
					t.kvRaw = append(t.kvRaw, '"')
					t.wsRaw = t.wsRaw[:0]
					f.ph = phColon
					i++
					continue
				}
				if !t.valueStart(f, KindString) {
					continue
				}
				if t.regOpen {
					rs = i
					if t.regInner {
						rs = i + 1
					}
				}
				t.st = sInStr
				t.esc = false
				i++
			case '{', '[':
				kind := KindObject
				if c == '[' {
					kind = KindArray
				}
				if !t.valueStart(f, kind) {
					continue
				}
				if t.regOpen {
					rs = i
				}
				t.depth++
				i++
			default:
				if !t.lit.start(c) {
					t.Bail("非法字符")
					continue
				}
				if !t.valueStart(f, t.lit.kind) {
					continue
				}
				if t.regOpen {
					rs = i
				}
				t.st = sInScalar
				i++
			}
		}
	}
	if rs >= 0 && t.regOpen && !t.dead {
		t.emitRegion(p[rs:])
	}
}

// flush 把 p[rs:end] 交给区域，返回新的 rs（-1）。
func (t *Transformer) flush(p []byte, rs, end int) int {
	if rs >= 0 && end > rs {
		t.emitRegion(p[rs:end])
	}
	return -1
}

// valueStart 在派发帧里一个值的第一个字节到达时决定动作。
// 返回 false 表示已 Bail。
func (t *Transformer) valueStart(f *frame, kind ValueKind) bool {
	if f.kind == fkObj {
		if f.ph != phValue {
			t.Bail("意外的值")
			return false
		}
		t.kvRaw = append(t.kvRaw, t.wsRaw...)
		t.wsRaw = t.wsRaw[:0]
	} else {
		if f.ph != phValue {
			t.Bail("数组元素之间缺少逗号")
			return false
		}
		// 数组元素开始
		f.idx++
		t.elemWs = append(t.elemWs[:0], t.wsRaw...)
		t.wsRaw = t.wsRaw[:0]
		t.path = append(t.path, seg{i: f.idx})
		t.pend = t.proto.OnElem(t)
		t.pendSet = true
		if t.dead {
			return false
		}
	}
	act := t.pend
	t.pendSet = false
	if act.kind == akProbe {
		act = t.proto.OnStart(t, kind)
		if t.dead {
			return false
		}
	}
	return t.apply(f, act, kind)
}

// apply 执行一个动作。
func (t *Transformer) apply(f *frame, act Action, kind ValueKind) bool {
	isContainer := kind == KindObject || kind == KindArray
	switch act.kind {
	case akBail:
		t.Bail(act.reason)
		return false
	case akProbe:
		t.Bail("Probe 不能嵌套")
		return false
	case akEnter:
		if !isContainer {
			if !act.lenient {
				t.Bail("期待容器值: " + t.PathString())
				return false
			}
			act.kind = akPass
			return t.apply(f, act, kind)
		}
		nf := frame{ph: phKey, idx: -1, flat: act.flat, lazy: act.lazy}
		if kind == KindArray {
			nf.kind = fkArr
			nf.ph = phValue
		}
		t.frames = append(t.frames, nf)
		if !act.flat {
			name := ""
			var raw []byte
			if f.kind == fkObj {
				name = act.key
				if name == "" {
					name = t.Last()
					raw = append([]byte(nil), t.kvRaw...)
				}
			} else {
				raw = append([]byte(nil), t.elemWs...)
			}
			t.w.push(name, raw, kind == KindArray)
		}
		return true
	case akPass, akObserve:
		if act.inner && kind != KindString {
			t.Bail("Inner 只能用于字符串值: " + t.PathString())
			return false
		}
		level := act.level
		if level < 0 {
			level = t.w.Level()
		}
		var ok bool
		if f.kind == fkObj {
			if act.key == "" {
				ok = t.w.KeyRawAt(level, t.kvRaw)
			} else {
				ok = t.w.KeyAt(level, act.key)
			}
		} else {
			ok = t.w.ElemRawAt(level, t.elemWs)
		}
		if !ok {
			t.Bail("目标输出层之上已有已打开的层: " + t.PathString())
			return false
		}
		if len(act.prefix) > 0 {
			t.w.Raw(act.prefix)
		}
		target := rtOut
		if act.kind == akObserve {
			target = rtObserve
		}
		t.beginRegion(target, act)
		return true
	case akSkip:
		t.beginRegion(rtSkip, act)
		return true
	case akCapture:
		t.beginRegion(rtCapture, act)
		return true
	case akDefer:
		if f.kind != fkObj {
			t.Bail("Defer 只能用于对象内的 key")
			return false
		}
		t.regKey = t.Last()
		t.beginRegion(rtDefer, act)
		return true
	case akPrefix:
		if kind != KindString {
			t.Bail("Prefix 只能用于字符串值: " + t.PathString())
			return false
		}
		act.inner = true
		t.beginRegion(rtPrefix, act)
		return true
	}
	t.Bail("未知动作")
	return false
}

func (t *Transformer) beginRegion(target regionTarget, act Action) {
	t.regOpen = true
	t.regT = target
	t.regDepth = t.depth
	t.regInner = act.inner
	t.regSuf = act.suffix
	t.regCap = act.cap
	t.capBuf = t.capBuf[:0]
}

// emitRegion 处理区域内的一段原始字节。
func (t *Transformer) emitRegion(b []byte) {
	switch t.regT {
	case rtOut:
		t.w.Raw(b)
	case rtSkip:
	case rtCapture, rtDefer:
		t.capAppend(b)
	case rtObserve:
		t.w.Raw(b)
		t.capAppend(b)
	case rtPrefix:
		room := t.regCap - len(t.capBuf)
		if room >= len(b) {
			t.capBuf = append(t.capBuf, b...)
			return
		}
		t.capBuf = append(t.capBuf, b[:room]...)
		rest := b[room:]
		t.runPrefix(false)
		if t.dead {
			return
		}
		// 窗口之后的字节按新目标处理
		t.emitRegion(rest)
	}
}

func (t *Transformer) capAppend(b []byte) {
	if t.regCap > 0 && len(t.capBuf)+len(b) > t.regCap {
		t.Bail("缓冲超过上限: " + t.PathString())
		return
	}
	t.capBuf = append(t.capBuf, b...)
}

// runPrefix 把前缀窗口交给协议，并按其返回切换区域目标。
func (t *Transformer) runPrefix(complete bool) {
	act, resume := t.proto.OnPrefix(t, t.capBuf, complete)
	if t.dead {
		return
	}
	if resume < 0 || resume > len(t.capBuf) {
		t.Bail("OnPrefix 返回了非法的 resume")
		return
	}
	switch act.kind {
	case akPass:
		if len(act.prefix) > 0 {
			t.w.Raw(act.prefix)
		}
		t.w.Raw(t.capBuf[resume:])
		t.regT = rtOut
		t.regSuf = act.suffix
	case akSkip:
		t.regT = rtSkip
		t.regSuf = nil
	case akBail:
		t.Bail(act.reason)
		return
	default:
		t.Bail("OnPrefix 只能返回 Pass / Skip / Bail")
		return
	}
	t.capBuf = t.capBuf[:0]
}

// endRegion 区域结束：交付缓冲、写后缀。
func (t *Transformer) endRegion() {
	if t.dead {
		return // 缓冲超限等 Bail 已发生，不再把残缺数据交给协议
	}
	switch t.regT {
	case rtOut:
		if len(t.regSuf) > 0 {
			t.w.Raw(t.regSuf)
		}
	case rtObserve:
		if len(t.regSuf) > 0 {
			t.w.Raw(t.regSuf)
		}
		t.proto.OnValue(t, t.capBuf)
	case rtCapture:
		t.proto.OnValue(t, t.capBuf)
	case rtDefer:
		f := t.top()
		raw := make([]byte, len(t.capBuf))
		copy(raw, t.capBuf)
		f.deferred = append(f.deferred, DeferredKV{Key: t.regKey, KeyRaw: append([]byte(nil), t.kvRaw...), Raw: raw})
	case rtPrefix:
		t.runPrefix(true)
		if t.dead {
			return
		}
		if t.regT == rtOut && len(t.regSuf) > 0 {
			t.w.Raw(t.regSuf)
		}
	}
	t.regOpen = false
	t.regT = rtNone
	t.regSuf = nil
	t.capBuf = t.capBuf[:0]
}

// onKeyDone key 闭合：派发 OnKey。
func (t *Transformer) onKeyDone() {
	var key string
	if t.keyEsc { // 带转义的 key：按 JSON 解码后再派发（原文仍由 kvRaw 保留）
		q := make([]byte, 0, len(t.keyBuf)+2)
		q = append(q, '"')
		q = append(q, t.keyBuf...)
		q = append(q, '"')
		if err := json.Unmarshal(q, &key); err != nil {
			t.Bail("key 转义非法")
			return
		}
	} else {
		key = string(t.keyBuf)
	}
	f := t.top()
	if t.DupKeyBail {
		for _, s := range f.seen {
			if s == key {
				t.Bail("重复的 key: " + key)
				return
			}
		}
		f.seen = append(f.seen, key)
	}
	t.path = append(t.path, seg{k: key, i: -1})
	t.pend = t.proto.OnKey(t)
	t.pendSet = true
	if t.pend.kind == akBail {
		t.Bail(t.pend.reason)
	}
}

// afterValue 一个值（标量 / 字符串 / 容器）在派发帧里结束。
func (t *Transformer) afterValue() {
	f := t.top()
	if f == nil || t.dead {
		return
	}
	f.ph = phComma
	f.n++
	if len(t.path) > 0 {
		t.path = t.path[:len(t.path)-1]
	}
	if t.wantRelease && t.releaseAt == len(t.frames)-1 {
		t.wantRelease = false
		t.doRelease()
	}
}

// closeContainer 派发帧闭合。
func (t *Transformer) closeContainer() {
	f := t.top()
	closeWs := append([]byte(nil), t.wsRaw...)
	t.wsRaw = t.wsRaw[:0]
	t.proto.OnLeave(t)
	if t.dead {
		return
	}
	if t.wantRelease && t.releaseAt == len(t.frames)-1 {
		t.wantRelease = false
		t.doRelease()
		if t.dead {
			return
		}
	}
	if len(f.deferred) > 0 {
		// 协议既没回放也没显式丢弃：这是协议逻辑漏洞，静默吞掉会产出语义不同的请求。
		t.Bail("容器闭合时仍有未处理的 Defer 项: " + t.PathString())
		return
	}
	flat, lazy := f.flat, f.lazy
	t.frames = t.frames[:len(t.frames)-1]
	t.depth--
	if t.depth == 0 {
		t.rootDone = true
		t.rootCloseWs = closeWs
		return // 根的输出层在 Finish 里闭合（Tail 之后）
	}
	if !flat {
		if !lazy {
			t.w.Open() // 输入里存在的容器，输出里也要存在，哪怕是空的
		}
		t.w.pop(closeWs)
	}
	t.afterValue()
}

// doRelease 回放当前派发帧里的 Defer 项：每一项重新经过 OnKey，按此刻的协议状态处理。
func (t *Transformer) doRelease() {
	f := t.top()
	if f == nil || len(f.deferred) == 0 {
		return
	}
	kvs := f.deferred
	f.deferred = nil
	for _, kv := range kvs {
		if t.dead {
			return
		}
		t.replayKV(kv)
	}
}

func (t *Transformer) replayKV(kv DeferredKV) {
	f := t.top()
	t.kvRaw = append(t.kvRaw[:0], kv.KeyRaw...)
	t.wsRaw = t.wsRaw[:0]
	t.path = append(t.path, seg{k: kv.Key, i: -1})
	t.pend = t.proto.OnKey(t)
	t.pendSet = true
	if t.pend.kind == akBail {
		t.Bail(t.pend.reason)
		return
	}
	f.ph = phValue
	t.scan(kv.Raw)
	if t.st == sInScalar {
		t.scan([]byte{' '}) // 补一个分隔符收尾标量
	}
	t.wsRaw = t.wsRaw[:0] // 上面的补位空格不属于原文
}

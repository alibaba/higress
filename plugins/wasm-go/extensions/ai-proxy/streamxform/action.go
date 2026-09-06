package streamxform

// 层 2 接口：协议 hooks 与动作。
//
// 协议实现是手写代码，不是规则表——协议之间差的是结构，不是字段名。
// 框架只提供"对一个 key/元素/值怎么处理"的动作集，决定权全在协议代码里。

type actKind uint8

const (
	akPass    actKind = iota // 规则 1：key+value 原样输出（可改名/加壳），不缓存
	akSkip                   // 规则 2：丢弃到该值结束，不缓存
	akCapture                // 规则 3-③：值完整缓冲后交给 OnValue 改写（只用于小值）
	akObserve                // Pass + 同时给 OnValue 一份拷贝（小值）
	akDefer                  // 有界暂存 key+value，等协议 Release 时按当时状态回放
	akEnter                  // 进入容器：子 key / 子元素继续交给协议决定
	akProbe                  // 先看值的类型再决定：回调 OnStart
	akPrefix                 // 字符串：先攒前缀窗口交给 OnPrefix，再决定后续
	akBail                   // 处理不了 → 回落
)

// Action 是协议对一个 key / 元素 / 值的处理决定。
// 用构造函数 + 链式修饰生成；零值无意义。
type Action struct {
	kind    actKind
	key     string   // Pass/Enter：输出时改名；"" 沿用原 key
	level   int      // -1 = 当前输出层；否则写到指定外层（要求其上所有层尚未打开）
	inner   bool     // Pass：只拷贝字符串内容（去掉两端引号）；非字符串则 Bail
	flat    bool     // Enter：不在输出侧建立对应层（协议自己决定写什么）
	lenient bool     // Enter：值不是容器时按 Pass 处理而不是 Bail
	lazy    bool     // Enter：闭合时若从未写入任何子项，就不物化（丢弃）；默认物化为 [] / {}
	prefix  []byte   // Pass：写在值前
	suffix  []byte   // Pass：写在值后
	cap     int      // Capture/Observe/Defer/Prefix：字节上限，0 = 不限
	reason  string   // Bail
	via     Protocol // Enter：这棵子树里的回调交给它（子 hook），容器闭合的 OnLeave 仍回到发起 Enter 的一方
}

func Pass() Action              { return Action{kind: akPass, level: -1} }
func Skip() Action              { return Action{kind: akSkip, level: -1} }
func Capture(cap int) Action    { return Action{kind: akCapture, level: -1, cap: cap} }
func Observe(cap int) Action    { return Action{kind: akObserve, level: -1, cap: cap} }
func Defer(cap int) Action      { return Action{kind: akDefer, level: -1, cap: cap} }
func Enter() Action             { return Action{kind: akEnter, level: -1} }
func Probe() Action             { return Action{kind: akProbe, level: -1} }
func Prefix(cap int) Action     { return Action{kind: akPrefix, level: -1, cap: cap} }
func Bail(reason string) Action { return Action{kind: akBail, level: -1, reason: reason} }

// As 改名（Pass / Enter）。
func (a Action) As(key string) Action { a.key = key; return a }

// At 写到指定输出层（Pass / Enter）。层号取自 Writer.Level()。
func (a Action) At(level int) Action { a.level = level; return a }

// Inner 只输出字符串内容，不带引号（Pass）。
func (a Action) Inner() Action { a.inner = true; return a }

// Wrap 在值两端加壳（Pass）。prefix/suffix 应是静态字节，回调期间不会被修改。
func (a Action) Wrap(prefix, suffix []byte) Action { a.prefix = prefix; a.suffix = suffix; return a }

// Flat：Enter 时不在输出侧建层（Enter）。
func (a Action) Flat() Action { a.flat = true; return a }

// Lazy：Enter 的容器闭合时若什么都没写，就一个字节都不输出（Enter）。
// 用于"元素可能整个被丢弃"的场合；默认行为是物化为空容器，与输入保持一致。
func (a Action) Lazy() Action { a.lazy = true; return a }

// Via：把这个容器内部的全部回调（OnKey / OnElem / OnStart / OnValue / OnPrefix / OnLeave、Defer 回放）
// 交给子 hook；子 hook 自己 Enter 的更深层也归它。容器本身闭合时的 OnLeave 回到发起 Enter 的一方，
// 让它收尾（如 Pop 自建的输出层）。路径与深度仍是绝对的。
func (a Action) Via(h Protocol) Action { a.via = h; return a }

// Lenient：Enter 遇到非容器值时退化为 Pass 而不是 Bail（Enter）。
func (a Action) Lenient() Action { a.lenient = true; return a }

// ValueKind 是 Probe 回调时告知协议的值类型。
type ValueKind uint8

const (
	KindString ValueKind = iota
	KindObject
	KindArray
	KindNull   // 字面量 null
	KindBool   // true / false
	KindNumber // 数字
)

// IsScalar 报告是否为标量（null / bool / number；字符串单列为 KindString）。
func (k ValueKind) IsScalar() bool { return k >= KindNull }

// IsContainer 报告是否为对象或数组。
func (k ValueKind) IsContainer() bool { return k == KindObject || k == KindArray }

func (k ValueKind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindObject:
		return "object"
	case KindArray:
		return "array"
	case KindNull:
		return "null"
	case KindBool:
		return "bool"
	case KindNumber:
		return "number"
	}
	return "?"
}

// Protocol 是一个目标协议的全部流式处理逻辑。
//
// 所有回调都在扫描过程中同步发生；raw 只在回调期间有效，要保留必须拷贝。
// 回调里可以通过 t 读取当前路径、写输出（t.W()）、请求回放（t.Release()）、
// 判定不支持（t.Bail()）。
type Protocol interface {
	// OnKey：扫到一个 key。t.Path 已包含该 key。
	OnKey(t *Transformer) Action
	// OnElem：扫到一个数组元素的开头。t.Path 已包含下标。
	OnElem(t *Transformer) Action
	// OnStart：Probe 之后，值的第一个字节到达，告知类型，要求最终动作。
	OnStart(t *Transformer, kind ValueKind) Action
	// OnValue：Capture / Observe 的值到齐。
	OnValue(t *Transformer, raw []byte)
	// OnPrefix：Prefix 的窗口攒满（complete=false）或字符串在窗口内就结束了（complete=true）。
	// raw 是字符串内容的原始字节（含转义、不含引号）。
	// 返回后续动作（Pass 或 Skip 或 Bail）与 resume：raw[resume:] 会紧接 prefix 之后输出。
	OnPrefix(t *Transformer, raw []byte, complete bool) (Action, int)
	// OnLeave：Enter 的容器闭合。t.Path 仍指向该容器。
	OnLeave(t *Transformer)
	// Tail：根对象闭合后、输出结尾 } 之前——规则 4 的"新增字段放最后"。
	Tail(t *Transformer)
}

// Prelude 是协议在扫描过程中收集到的、集成层需要用来产生副作用的事实
// （请求头、上下文键）。字段"是否已见"与"值"分开：没见到不等于 false。
type Prelude struct {
	Model      string
	ModelSeen  bool
	Stream     bool
	StreamSeen bool
}

// Preluder 由需要向集成层报告 Prelude 的协议实现。
type Preluder interface {
	Prelude() Prelude
}

// BaseProtocol 给出全部回调的空实现（一律 Pass / 不动作）。
// 协议或子 hook 嵌入它之后只需覆盖自己关心的回调，新协议从"什么都直通"起步。
type BaseProtocol struct{}

func (BaseProtocol) OnKey(*Transformer) Action                         { return Pass() }
func (BaseProtocol) OnElem(*Transformer) Action                        { return Pass() }
func (BaseProtocol) OnStart(*Transformer, ValueKind) Action            { return Pass() }
func (BaseProtocol) OnValue(*Transformer, []byte)                      {}
func (BaseProtocol) OnPrefix(*Transformer, []byte, bool) (Action, int) { return Pass(), 0 }
func (BaseProtocol) OnLeave(*Transformer)                              {}
func (BaseProtocol) Tail(*Transformer)                                 {}

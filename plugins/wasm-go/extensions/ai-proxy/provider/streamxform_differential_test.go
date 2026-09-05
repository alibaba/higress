package provider

// 流式请求体转换（streamxform）的差分测试：同一批输入喂给官方实现与流式实现，逐字段比对。
//
// 官方侧尽量直接调用本包的纯函数（buildClaudeTextGenRequest、convertDeveloperRoleToSystem、
// getMappedModel …）；依赖宿主的分支（defaultTransformRequestBody 里的请求头、Claude 上下文）
// 在集成层已判定不走流式，这里按官方代码逐行复刻其纯函数部分。
// 解码用 UseNumber：数字字面量也参与比对（1e0 与 1 不同），官方对读取的数字字段会重新格式化。
//
// 随机模糊默认跑较小规模以适应 CI；设置 STREAMXFORM_FUZZ_N 可放大。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/streamxform"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func fuzzN(def int) int {
	if v := os.Getenv("STREAMXFORM_FUZZ_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// ---- 来源: difftest_test.go ----
// officialClaude 复刻官方 claude provider 在 handleRequestBody 里的完整纯函数路径：
// developer→system 转换 → decodeChatCompletionRequest → mapModel → buildClaudeTextGenRequest。
func officialClaude(in string, claudeCode bool) (map[string]any, bool) {
	m, err := officialClaudeErr(in, claudeCode)
	return m, err == nil
}

func officialClaudeErr(in string, claudeCode bool) (map[string]any, error) {
	body, err := convertDeveloperRoleToSystem([]byte(in))
	if err != nil {
		return nil, err
	}
	req := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, req); err != nil {
		return nil, err
	}
	if req.Model == "" {
		return nil, fmt.Errorf("missing model in request") // mapModel
	}
	c := &claudeProvider{config: ProviderConfig{claudeCodeMode: claudeCode}}
	cr := c.buildClaudeTextGenRequest(req)
	b, err := json.Marshal(cr)
	if err != nil {
		return nil, err
	}
	m, err := decodeMap(b)
	return m, err
}

func officialTransform(t *testing.T, in string) (map[string]any, bool) {
	return officialClaude(in, false)
}

func runStream(tr *streamxform.Transformer, in string, chunk int) (map[string]any, bool, string) {
	var out []byte
	for i := 0; i < len(in); i += chunk {
		j := i + chunk
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		out = append(out, tr.Out()...)
	}
	out = append(out, tr.Finish()...)
	if bad, why := tr.Unsupported(); bad {
		return nil, false, why
	}
	m, err := decodeMap(out)
	if err != nil {
		return nil, false, "输出非法 JSON: " + err.Error() + " :: " + string(out)
	}
	return m, true, ""
}

func streamTransform(in string, chunk int) (map[string]any, bool, string) {
	return runStream(streamxform.New(), in, chunk)
}

var diffCases = []struct{ name, in string }{
	{"最简", `{"model":"m","messages":[{"role":"user","content":"U"}]}`},
	{"带system", `{"model":"m","messages":[{"role":"system","content":"S"},{"role":"user","content":"U"}]}`},
	{"多轮", `{"model":"m","messages":[{"role":"user","content":"A"},{"role":"assistant","content":"B"},{"role":"user","content":"C"}]}`},
	{"全标量", `{"model":"m","max_tokens":100,"stream":true,"temperature":0.7,"top_p":0.9,"stop":["END","X"],"messages":[{"role":"user","content":"U"}]}`},
	{"max_completion优先", `{"model":"m","max_tokens":10,"max_completion_tokens":200,"messages":[{"role":"user","content":"U"}]}`},
	{"Claude无对应字段", `{"model":"m","seed":42,"n":1,"user":"u","presence_penalty":0.5,"messages":[{"role":"user","content":"U"}]}`},
	{"乱序", `{"messages":[{"content":"U","role":"user"}],"stop":["E"],"model":"m","max_tokens":50}`},
	{"缺max_tokens", `{"model":"m","messages":[{"role":"user","content":"U"}]}`},
	{"转义内容", `{"model":"m","messages":[{"role":"user","content":"a\"b\\c\nd<>&"}]}`},
	{"零值标量", `{"model":"m","temperature":0,"top_p":0.0,"max_tokens":0,"stream":false,"messages":[{"role":"user","content":"U"}]}`},
	{"空system", `{"model":"m","messages":[{"role":"system","content":""},{"role":"user","content":"U"}]}`},
	{"两个system", `{"model":"m","messages":[{"role":"system","content":"S1"},{"role":"user","content":"U"},{"role":"system","content":"S2"}]}`},
	{"system数组", `{"model":"m","messages":[{"role":"system","content":[{"type":"text","text":"S1"},{"type":"text","text":"S2"}]},{"role":"user","content":"U"}]}`},
	{"全是system", `{"model":"m","messages":[{"role":"system","content":"S"}]}`},
	{"developer", `{"model":"m","messages":[{"role":"developer","content":"D"},{"role":"user","content":"U"}]}`},
	{"role缺失", `{"model":"m","messages":[{"content":"U"}]}`},
	{"空对象消息", `{"model":"m","messages":[{}]}`},
	{"content null", `{"model":"m","messages":[{"role":"assistant","content":null}]}`},
	{"content 缺失", `{"model":"m","messages":[{"role":"user"}]}`},
	{"content 数字", `{"model":"m","messages":[{"role":"user","content":42}]}`},
	{"content 对象", `{"model":"m","messages":[{"role":"user","content":{"a":1}}]}`},
	{"多模态文本", `{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"U1"},{"type":"text","text":"U2"}]}]}`},
	{"多模态空数组", `{"model":"m","messages":[{"role":"user","content":[]}]}`},
	{"多模态type迟到", `{"model":"m","messages":[{"role":"user","content":[{"text":"U1","type":"text"}]}]}`},
	{"多模态未知type", `{"model":"m","messages":[{"role":"user","content":[{"type":"video","text":"x"},{"type":"text","text":"U"}]}]}`},
	{"多模态缺type", `{"model":"m","messages":[{"role":"user","content":[{"text":"x"},{"type":"text","text":"U"}]}]}`},
	{"多模态非对象元素", `{"model":"m","messages":[{"role":"user","content":["str",1,{"type":"text","text":"U"}]}]}`},
	{"图片data", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo=","detail":"high"}}]}]}`},
	{"图片data转义", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image\/jpeg;base64,\/9j\/4AAQ"}}]}]}`},
	{"图片data非base64", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:text/plain;charset=utf8,hello"}}]}]}`},
	{"图片data空", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,"}}]}]}`},
	{"图片url", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://x/y.png"}}]}]}`},
	{"图片url空", `{"model":"m","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":""}}]}]}`},
	{"图片type迟到", `{"model":"m","messages":[{"role":"user","content":[{"image_url":{"url":"https://x/y.png"},"type":"image_url"}]}]}`},
	{"文件", `{"model":"m","messages":[{"role":"user","content":[{"type":"file","file":{"file_id":"f1","file_name":"a.pdf"}}]}]}`},
	{"文件空id", `{"model":"m","messages":[{"role":"user","content":[{"type":"file","file":{"file_id":""}}]}]}`},
	{"音频跳过", `{"model":"m","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"AA","format":"wav"}},{"type":"text","text":"U"}]}]}`},
	{"cache_control", `{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"U","cache_control":{"type":"ephemeral"}}]}]}`},
	{"tools", `{"model":"m","tools":[{"type":"function","function":{"name":"f","description":"d","parameters":{"type":"object","properties":{"a":{"type":"string"}}}}}],"messages":[{"role":"user","content":"U"}]}`},
	{"tools无参数", `{"model":"m","tools":[{"type":"function","function":{"name":"f","parameters":{}}},{"type":"function","function":{"name":"g"}}],"messages":[{"role":"user","content":"U"}]}`},
	{"tools空", `{"model":"m","tools":[],"messages":[{"role":"user","content":"U"}]}`},
	{"tools null", `{"model":"m","tools":null,"messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice auto", `{"model":"m","tool_choice":"auto","messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice none", `{"model":"m","tool_choice":"none","messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice required", `{"model":"m","tool_choice":"required","parallel_tool_calls":false,"messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice function", `{"model":"m","tool_choice":{"type":"function","function":{"name":"f"}},"messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice function无名", `{"model":"m","tool_choice":{"type":"function","function":{}},"messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice null", `{"model":"m","tool_choice":null,"messages":[{"role":"user","content":"U"}]}`},
	{"tool_choice+thinking", `{"model":"m","tool_choice":"required","reasoning_effort":"low","messages":[{"role":"user","content":"U"}]}`},
	{"tool_calls", `{"model":"m","messages":[{"role":"assistant","content":"x","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{\"a\":1}"}}]}]}`},
	{"tool_calls空content", `{"model":"m","messages":[{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":""}}]}]}`},
	{"tool_calls null content", `{"model":"m","messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"bad json"}}]}]}`},
	{"tool_calls先于content", `{"model":"m","messages":[{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}],"content":"x"}]}`},
	{"tool_calls空数组", `{"model":"m","messages":[{"role":"assistant","content":"x","tool_calls":[]}]}`},
	{"tool_calls多个", `{"model":"m","messages":[{"role":"assistant","content":"x","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}},{"id":"c2","type":"function","function":{"name":"g","arguments":"null"}}]}]}`},
	{"tool_calls在user上", `{"model":"m","messages":[{"role":"user","content":"x","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]}]}`},
	{"tool role", `{"model":"m","messages":[{"role":"tool","tool_call_id":"c1","content":"r1"}]}`},
	{"tool role合并", `{"model":"m","messages":[{"role":"user","content":"U"},{"role":"tool","tool_call_id":"c1","content":"r1"},{"role":"tool","tool_call_id":"c2","content":"r2"},{"role":"assistant","content":"A"}]}`},
	{"tool role隔system", `{"model":"m","messages":[{"role":"tool","tool_call_id":"c1","content":"r1"},{"role":"system","content":"S"},{"role":"tool","tool_call_id":"c2","content":"r2"}]}`},
	{"tool role数组", `{"model":"m","messages":[{"role":"tool","tool_call_id":"c1","content":[{"type":"text","text":"a"},{"type":"text","text":"b"}]}]}`},
	{"tool role无id", `{"model":"m","messages":[{"role":"tool","content":"r1"}]}`},
	{"tool role迟到", `{"model":"m","messages":[{"content":"r1","tool_call_id":"c1","role":"tool"}]}`},
	{"tool role末尾", `{"model":"m","messages":[{"role":"user","content":"U"},{"role":"tool","tool_call_id":"c1","content":"r1"}]}`},
	{"reasoning_effort", `{"model":"m","reasoning_effort":"high","max_tokens":20000,"messages":[{"role":"user","content":"U"}]}`},
	{"reasoning小于max", `{"model":"m","reasoning_effort":"high","max_tokens":2000,"messages":[{"role":"user","content":"U"}]}`},
	{"reasoning太小", `{"model":"m","reasoning_effort":"low","max_tokens":500,"messages":[{"role":"user","content":"U"}]}`},
	{"reasoning_max_tokens", `{"model":"m","reasoning_max_tokens":3000,"messages":[{"role":"user","content":"U"}]}`},
	{"claude_thinking", `{"model":"m","claude_thinking":{"type":"enabled","budget_tokens":2048},"reasoning_effort":"high","messages":[{"role":"user","content":"U"}]}`},
	{"claude_thinking disabled", `{"model":"m","claude_thinking":{"type":"disabled"},"tool_choice":"required","messages":[{"role":"user","content":"U"}]}`},
	{"claude_output_config", `{"model":"m","claude_output_config":{"effort":"low","format":{"type":"json"}},"messages":[{"role":"user","content":"U"}]}`},
	{"claude_anthropic_beta", `{"model":"m","claude_anthropic_beta":["x"],"messages":[{"role":"user","content":"U"}]}`},
	{"stop空", `{"model":"m","stop":[],"messages":[{"role":"user","content":"U"}]}`},
	{"stop null", `{"model":"m","stop":null,"messages":[{"role":"user","content":"U"}]}`},
	{"温度null", `{"model":"m","temperature":null,"messages":[{"role":"user","content":"U"}]}`},
	{"温度整数", `{"model":"m","temperature":1,"top_p":1e0,"messages":[{"role":"user","content":"U"}]}`},
	{"负max_tokens", `{"model":"m","max_tokens":-5,"messages":[{"role":"user","content":"U"}]}`},
	{"message未知字段", `{"model":"m","messages":[{"role":"user","content":"U","name":"n","reasoning_content":"r","refusal":"x","zzz":{"a":[1,2]}}]}`},
	{"顶层嵌套未知", `{"model":"m","logit_bias":{"1":2},"metadata":{"k":"v"},"response_format":{"type":"json_schema","json_schema":{"schema":{"content":1}}},"messages":[{"role":"user","content":"U"}]}`},
	{"空白格式化", "{\n  \"model\" : \"m\" ,\n  \"messages\" : [ { \"role\" : \"user\" , \"content\" : \"U\" } ]\n}\n"},
	{"unicode转义", `{"model":"mA","messages":[{"role":"user","content":"中😀"}]}`},
}

func TestDifferential(t *testing.T) {
	fmt.Println("  用例                     官方 vs 流式")
	fmt.Println("  ----------------------   ------------------------------------")
	for _, c := range diffCases {
		for _, cc := range []bool{false, true} {
			off, offOK := officialClaude(c.in, cc)
			for _, cs := range []int{1, 7, 4096} {
				str, ok, why := runStream(streamxform.NewClaude(streamxform.ClaudeOptions{ClaudeCodeMode: cc}), c.in, cs)
				if !offOK {
					if ok {
						t.Errorf("%s(cc=%v): 官方失败但流式放行了", c.name, cc)
					} else if cs == 4096 && !cc {
						fmt.Printf("  %-22s   官方失败，流式回落 ✓ (%s)\n", c.name, why)
					}
					continue
				}
				if !ok {
					t.Errorf("%s(cc=%v) chunk=%d: 官方成功但流式回落: %s", c.name, cc, cs, why)
					continue
				}
				if d := diffMaps(off, str); d != "" {
					t.Errorf("%s(cc=%v) chunk=%d 与官方不一致:\n    %s", c.name, cc, cs, d)
				} else if cs == 4096 && !cc {
					fmt.Printf("  %-22s   ✓ 一致\n", c.name)
				}
			}
		}
	}
}

func diffMaps(a, b map[string]any) string {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) == string(jb) {
		return ""
	}
	// 注意：map[string]any 里数字都成了 float64，1e0 与 1 在这里分不出来；
	// 网关差分用原始字节比对，抓到过这类"数值相等、字面不同"的差异（已改为与官方同样的格式化）。
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	out := ""
	for k := range keys {
		va, _ := json.Marshal(a[k])
		vb, _ := json.Marshal(b[k])
		if string(va) != string(vb) {
			out += fmt.Sprintf("[%s] 官方=%s 流式=%s  ", k, va, vb)
		}
	}
	return out
}

// decodeMap 用 UseNumber 解码：数字保留字面量，1e0 与 1 视为不同——官方对读取的数字字段会经 float64 重新格式化，
// 流式必须走同样的格式化，否则字节不同（网关差分抓到过）。
func decodeMap(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// ---- 来源: fuzz_test.go ----
// 随机生成 OpenAI 请求：覆盖 buildClaudeTextGenRequest 读取的每一个字段，
// 字段顺序、取值、消息形态全随机。
func genRequest(r *rand.Rand) string {
	type kv struct{ k, v string }
	var fields []kv
	add := func(k, v string) { fields = append(fields, kv{k, v}) }
	pick := func(xs ...string) string { return xs[r.Intn(len(xs))] }
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) }

	if r.Intn(30) != 0 {
		add("model", q("m"+fmt.Sprint(r.Intn(5))))
	}
	if r.Intn(2) == 0 {
		add("max_tokens", pick(fmt.Sprint(r.Intn(30000)+1), "0", "-3"))
	}
	if r.Intn(3) == 0 {
		add("max_completion_tokens", pick(fmt.Sprint(r.Intn(30000)+1), "0"))
	}
	if r.Intn(2) == 0 {
		add("stream", pick("true", "false"))
	}
	if r.Intn(2) == 0 {
		add("temperature", pick(fmt.Sprintf("%.2f", r.Float64()), "0", "1", "0.0", "1e0"))
	}
	if r.Intn(2) == 0 {
		add("top_p", pick(fmt.Sprintf("%.2f", r.Float64()), "0", "null"))
	}
	if r.Intn(3) == 0 {
		n := r.Intn(3)
		var ss []string
		for i := 0; i < n; i++ {
			ss = append(ss, q(randText(r)))
		}
		add("stop", pick("["+strings.Join(ss, ",")+"]", "null"))
	}
	for _, f := range []string{"seed", "n", "presence_penalty", "frequency_penalty", "top_logprobs"} {
		if r.Intn(5) == 0 {
			add(f, fmt.Sprint(r.Intn(100)))
		}
	}
	if r.Intn(5) == 0 {
		add("logprobs", "true")
	}
	if r.Intn(5) == 0 {
		add("user", q(randText(r)))
	}
	if r.Intn(5) == 0 {
		add("metadata", `{"k":`+q(randText(r))+`,"model":"trap","messages":[{"role":"x"}]}`)
	}
	if r.Intn(5) == 0 {
		add("response_format", `{"type":"json_schema","json_schema":{"schema":{"content":{"role":1}}}}`)
	}
	if r.Intn(6) == 0 {
		add("stream_options", `{"include_usage":true}`)
	}
	if r.Intn(8) == 0 {
		add("unknown_"+fmt.Sprint(r.Intn(3)), pick(`1`, `"s"`, `{"a":[1,{"b":null}]}`, `[]`))
	}
	// reasoning / thinking
	if r.Intn(4) == 0 {
		add("reasoning_effort", q(pick("low", "medium", "high", "xyz", "")))
	}
	if r.Intn(5) == 0 {
		add("reasoning_max_tokens", pick(fmt.Sprint(r.Intn(40000)), "0"))
	}
	if r.Intn(6) == 0 {
		add("claude_thinking", pick(`{"type":"enabled","budget_tokens":`+fmt.Sprint(r.Intn(20000))+`}`, `{"type":"disabled"}`, `{"type":""}`, `null`, `{"type":"enabled","display":"summarized"}`))
	}
	if r.Intn(8) == 0 {
		add("claude_output_config", pick(`{"effort":"low"}`, `{"format":{"type":"json_schema","schema":{"a":1}}}`, `{"effort":"","format":null}`, `{}`))
	}
	if r.Intn(8) == 0 {
		add("claude_anthropic_beta", `["b1"]`)
	}
	// tools
	if r.Intn(3) == 0 {
		n := r.Intn(3)
		var ts []string
		for i := 0; i < n; i++ {
			fn := `"name":` + q("f"+fmt.Sprint(i))
			if r.Intn(2) == 0 {
				fn += `,"description":` + q(randText(r))
			}
			switch r.Intn(4) {
			case 0:
				fn += `,"parameters":{}`
			case 1:
				fn += `,"parameters":{"type":"object","properties":{"z":{"type":"string"},"a":{"type":"number","enum":[1,2.5]}},"required":["a"]}`
			case 2:
				fn += `,"parameters":null`
			}
			ts = append(ts, `{"type":"function","function":{`+fn+`}}`)
		}
		add("tools", pick("["+strings.Join(ts, ",")+"]", "null"))
	}
	if r.Intn(3) == 0 {
		add("tool_choice", pick(`"auto"`, `"none"`, `"required"`, `"function"`, `{"type":"function","function":{"name":"f1"}}`, `{"type":"function","function":{}}`, `{"type":"auto"}`, `null`, `{"type":"tool","function":{"name":"g"}}`))
	}
	if r.Intn(4) == 0 {
		add("parallel_tool_calls", pick("true", "false", "null"))
	}

	// messages
	nmsg := r.Intn(5)
	if r.Intn(20) != 0 && nmsg == 0 {
		nmsg = 1
	}
	var msgs []string
	for i := 0; i < nmsg; i++ {
		msgs = append(msgs, genMessage(r))
	}
	add("messages", pick("["+strings.Join(msgs, ",")+"]", "["+strings.Join(msgs, ",")+"]", "["+strings.Join(msgs, ",")+"]", "null"))

	r.Shuffle(len(fields), func(i, j int) { fields[i], fields[j] = fields[j], fields[i] })
	var out []string
	for _, f := range fields {
		out = append(out, `"`+f.k+`":`+f.v)
	}
	return "{" + strings.Join(out, ",") + "}"
}

func genMessage(r *rand.Rand) string {
	type kv struct{ k, v string }
	var parts []kv
	add := func(k, v string) { parts = append(parts, kv{k, v}) }
	pick := func(xs ...string) string { return xs[r.Intn(len(xs))] }
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) }

	role := pick("user", "user", "assistant", "assistant", "system", "developer", "tool", "tool", "")
	if r.Intn(12) != 0 {
		add("role", q(role))
	} else {
		role = ""
	}
	// content
	switch r.Intn(8) {
	case 0:
		// 缺失
	case 1:
		add("content", "null")
	case 2:
		add("content", genParts(r))
	case 3:
		add("content", pick("42", "true", `{"a":1}`))
	default:
		add("content", q(randText(r)))
	}
	if role == "assistant" && r.Intn(2) == 0 {
		n := r.Intn(3)
		var tcs []string
		for i := 0; i < n; i++ {
			args := pick(`"{\"a\":1}"`, `""`, `"not json"`, `"null"`, `"{}"`, q(`{"k":"`+randText(r)+`"}`))
			tc := `{"id":` + q("c"+fmt.Sprint(i)) + `,"type":"function","function":{"name":` + q("f"+fmt.Sprint(i)) + `,"arguments":` + args + `}}`
			if r.Intn(4) == 0 {
				tc = `{"type":"function","function":{"name":"f","arguments":"{}"}}`
			}
			tcs = append(tcs, tc)
		}
		add("tool_calls", pick("["+strings.Join(tcs, ",")+"]", "null"))
	}
	if role == "tool" && r.Intn(5) != 0 {
		add("tool_call_id", q("call_"+fmt.Sprint(r.Intn(9))))
	}
	if r.Intn(6) == 0 {
		add("name", q("n"))
	}
	if r.Intn(8) == 0 {
		add("reasoning_content", q(randText(r)))
	}
	if r.Intn(10) == 0 {
		add("zz_unknown", `{"role":"system","content":[{"type":"text"}]}`)
	}
	r.Shuffle(len(parts), func(i, j int) { parts[i], parts[j] = parts[j], parts[i] })
	var out []string
	for _, p := range parts {
		out = append(out, `"`+p.k+`":`+p.v)
	}
	return "{" + strings.Join(out, ",") + "}"
}

func genParts(r *rand.Rand) string {
	pick := func(xs ...string) string { return xs[r.Intn(len(xs))] }
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	n := r.Intn(4)
	var ps []string
	for i := 0; i < n; i++ {
		var kvs []string
		switch r.Intn(7) {
		case 0, 1:
			kvs = []string{`"type":"text"`, `"text":` + q(randText(r))}
		case 2:
			url := pick("data:image/png;base64,"+randB64(r), "data:image\\/jpeg;base64,"+randB64(r), "https://x/"+randB64(r), "data:text/plain,hi", "data:nosemicolon", "", "data:;base64,"+randB64(r), "data:image/png;charset=x;base64,"+randB64(r))
			img := `{"url":"` + url + `"`
			if r.Intn(2) == 0 {
				img += `,"detail":"high"`
			}
			img += `}`
			kvs = []string{`"type":"image_url"`, `"image_url":` + img}
		case 3:
			kvs = []string{`"type":"file"`, `"file":` + pick(`{"file_id":"f1"}`, `{"file_id":""}`, `{"file_id":"f2","file_name":"a.pdf"}`)}
		case 4:
			kvs = []string{`"type":"input_audio"`, `"input_audio":{"data":"AA","format":"wav"}`}
		case 5:
			kvs = []string{`"type":` + pick(`"video"`, `1`, `null`), `"text":"x"`}
		case 6:
			kvs = []string{`"text":"x"`} // 缺 type
		}
		if r.Intn(4) == 0 {
			kvs = append(kvs, `"cache_control":{"type":"ephemeral"}`)
		}
		if r.Intn(6) == 0 {
			kvs = append(kvs, `"text":1`) // 类型不对
		}
		r.Shuffle(len(kvs), func(i, j int) { kvs[i], kvs[j] = kvs[j], kvs[i] })
		// 重复 key 会让流式 Bail（官方后者覆盖），这里去重以专注比对语义
		seen := map[string]bool{}
		var uniq []string
		for _, kv := range kvs {
			k := kv[:strings.Index(kv, ":")]
			if seen[k] {
				continue
			}
			seen[k] = true
			uniq = append(uniq, kv)
		}
		ps = append(ps, "{"+strings.Join(uniq, ",")+"}")
	}
	if r.Intn(6) == 0 {
		ps = append(ps, pick(`"str"`, `1`, `null`, `[]`))
	}
	return "[" + strings.Join(ps, ",") + "]"
}

func randB64(r *rand.Rand) string {
	const a = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	n := r.Intn(600)
	b := make([]byte, n)
	for i := range b {
		b[i] = a[r.Intn(len(a))]
	}
	return string(b)
}

func randText(r *rand.Rand) string {
	alph := []string{"a", "b", "中", "文", `"`, `\`, "\n", "\t", "é", " ", "😀", "<", "&", " ", "\x01"}
	n := r.Intn(40)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(alph[r.Intn(len(alph))])
	}
	return sb.String()
}

func TestFuzzDifferential(t *testing.T) {
	seed := int64(time.Now().UnixNano())
	r := rand.New(rand.NewSource(seed))
	N := fuzzN(20000)
	same, fallback, offFail, lenient := 0, 0, 0, 0
	for i := 0; i < N; i++ {
		in := genRequest(r)
		cc := r.Intn(4) == 0
		off, oerr := officialClaudeErr(in, cc)
		ok := oerr == nil
		chunk := []int{1, 3, 17, 64, 4096}[r.Intn(5)]
		str, sok, why := runStream(streamxform.NewClaude(streamxform.ClaudeOptions{ClaudeCodeMode: cc}), in, chunk)
		if !ok {
			offFail++
			if sok {
				// 唯一允许的宽松：官方 struct 对"被丢弃字段"的类型校验失败（如 metadata 里塞了数组）。
				// 流式不读这些字段，产出的请求与官方在该字段合法时的输出相同。
				if strings.Contains(oerr.Error(), "cannot unmarshal") {
					lenient++
					continue
				}
				t.Fatalf("第 %d 例官方失败但流式放行 (chunk=%d): %v\n  输入: %s", i, chunk, oerr, in)
			}
			continue
		}
		if !sok {
			fallback++
			if !strings.Contains(why, "重复的 key") && !strings.Contains(why, "官方会跳过") && !strings.Contains(why, "panic") {
				t.Fatalf("第 %d 例意外回落 (chunk=%d): %s\n  输入: %s", i, chunk, why, in)
			}
			continue
		}
		if d := diffMaps(off, str); d != "" {
			t.Fatalf("第 %d 例不一致 (chunk=%d cc=%v)\n  输入: %s\n  差异: %s", i, chunk, cc, in, d)
		}
		same++
	}
	fmt.Printf("  随机 %d 例 (seed=%d): 一致 %d, 官方失败 %d (其中丢弃字段类型错误、流式放行 %d), 已知回落 %d, 不一致 0\n", N, seed, same, offFail, lenient, fallback)
}

// ---- 来源: openai_diff_test.go ----
// officialOpenAI 复刻默认路径的纯函数部分（main.go normalizeOpenAiRequestBody →
// handleRequestBody: convertDeveloperRoleToSystem → defaultTransformRequestBody 的 model 改写）。
func officialOpenAI(in string, mapping map[string]string, chat bool) (map[string]any, bool) {
	body := []byte(in)
	if chat {
		var err error
		if gjson.GetBytes(body, "stream").Bool() && (!gjson.GetBytes(body, "stream_options").Exists() || !gjson.GetBytes(body, "stream_options.include_usage").Exists()) {
			body, err = sjson.SetBytes(body, "stream_options.include_usage", true)
			if err != nil {
				return nil, false
			}
		}
		body, err = convertDeveloperRoleToSystem(body)
		if err != nil {
			return nil, false
		}
	}
	model := gjson.GetBytes(body, "model").String()
	body, err := sjson.SetBytes(body, "model", getMappedModel(model, mapping))
	if err != nil {
		return nil, false
	}
	return toMap(body)
}

func hasDeveloper(in string) bool {
	for _, m := range gjson.Get(in, "messages").Array() {
		if m.Get("role").String() == "developer" {
			return true
		}
	}
	return false
}

var oaiMapping = map[string]string{"m1": "mapped-1", "gpt-*": "g", "~^re(.*)$": "x$1", "*": "fallback"}

func TestOpenAIDifferential(t *testing.T) {
	cases := []string{
		`{"model":"m1","messages":[{"role":"user","content":"U"}]}`,
		`{"messages":[{"role":"user","content":"U"}],"model":"gpt-4o","stream":true}`,
		`{"model":"re9","stream":true,"stream_options":{"x":1},"messages":[{"role":"user","content":"U"}]}`,
		`{"model":"zzz","stream":true,"stream_options":{"include_usage":false},"messages":[{"role":"user","content":"U"}]}`,
		`{"model":"m1","stream":false,"messages":[{"role":"user","content":"U"}]}`,
		`{"model":"m1","stream":"true","messages":[{"role":"user","content":"U"}]}`,
		`{"model":"m1","stream":1,"messages":[{"role":"user","content":"U"}]}`,
		`{"messages":[{"role":"user","content":"U"}]}`,
		`{"model":"m1","messages":[{"role":"user","content":[{"type":"text","text":"a"}]}],"tools":[{"x":{"y":[1,2,{"z":null}]}}],"anything":"goes","nested":{"model":"inner"}}`,
		`{"model":"m1","messages":[{"content":"U","role":"user","name":"n"}],"stream":true,"stream_options":{}}`,
		"{ \"model\" : \"m1\" ,\n \"messages\" : [ ] }",
		`{"model":"m1","messages":"not-an-array"}`,
		`{"model":"m1","messages":[1,"s",null,{"role":"user"}]}`,
		`{"model":"m1","messages":[{"role":"developer","content":"D"}]}`,
		`{"model":"m1","model":"dup","messages":[]}`,
	}
	fmt.Println("  OpenAI 透传用例                                   官方 vs 流式")
	for _, in := range cases {
		for _, chat := range []bool{true, false} {
			off, ok := officialOpenAI(in, oaiMapping, chat)
			if !ok {
				// 官方在 chat 路径会把 body 整体 Unmarshal（developer 转换），类型不合法即失败；
				// 流式透传不做校验，交给上游拒绝——已知且记录在案的宽松差异。
				if chat {
					fmt.Printf("  %-48s   官方 struct 校验失败，流式透传（已知宽松差异）\n", trunc(in, 48))
				}
				continue
			}
			for _, cs := range []int{1, 5, 4096} {
				tr := streamxform.NewOpenAI(streamxform.OpenAIOptions{
					MapModel:       func(m string) string { return getMappedModel(m, oaiMapping) },
					DetectStream:   chat,
					NormalizeUsage: chat,
					CheckMessages:  chat,
				})
				str, sok, why := runStream(tr, in, cs)
				if !sok {
					if chat && hasDeveloper(in) {
						if cs == 4096 {
							fmt.Printf("  %-48s   developer → 回落 ✓\n", trunc(in, 48))
						}
						continue
					}
					t.Errorf("chunk=%d chat=%v 意外回落: %s\n  %s", cs, chat, why, in)
					continue
				}
				if d := diffMaps(off, str); d != "" {
					t.Errorf("chunk=%d chat=%v 不一致: %s\n  输入: %s", cs, chat, d, in)
				} else if cs == 4096 && chat {
					fmt.Printf("  %-48s   ✓ 一致\n", trunc(in, 48))
				}
			}
		}
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func TestOpenAIFuzz(t *testing.T) {
	seed := int64(time.Now().UnixNano())
	r := rand.New(rand.NewSource(seed))
	N := fuzzN(10000)
	same, fb := 0, 0
	for i := 0; i < N; i++ {
		in := genRequest(r)
		if r.Intn(3) == 0 {
			in = strings.Replace(in, `"model":`, `"stream":`+[]string{"true", "false", `"true"`, "1"}[r.Intn(4)]+`,"model":`, 1)
		}
		chat := r.Intn(4) != 0
		off, ok := officialOpenAI(in, oaiMapping, chat)
		if !ok {
			continue
		}
		chunk := []int{1, 3, 17, 64, 4096}[r.Intn(5)]
		tr := streamxform.NewOpenAI(streamxform.OpenAIOptions{
			MapModel:       func(m string) string { return getMappedModel(m, oaiMapping) },
			DetectStream:   chat,
			NormalizeUsage: chat,
			CheckMessages:  chat,
		})
		str, sok, why := runStream(tr, in, chunk)
		if !sok {
			if chat && hasDeveloper(in) {
				fb++
				continue
			}
			t.Fatalf("第 %d 例意外回落 (chunk=%d chat=%v): %s\n  输入: %s", i, chunk, chat, why, in)
		}
		if d := diffMaps(off, str); d != "" {
			t.Fatalf("第 %d 例不一致 (chunk=%d chat=%v)\n  输入: %s\n  差异: %s", i, chunk, chat, in, d)
		}
		same++
	}
	fmt.Printf("  OpenAI 透传随机 %d 例 (seed=%d): 一致 %d, developer 回落 %d, 不一致 0\n", N, seed, same, fb)
}

// ---- 来源: variants_diff_test.go ----
// 官方各 provider 的纯函数路径（main.go normalize → handleRequestBody 前置步骤 → provider 特有逻辑 → 默认 model 改写）。
// 逐行照抄官方代码，只把依赖宿主的 ctx 分支去掉（那些分支在集成层已判不适用）。

func oaiPre(in string, chat bool) ([]byte, bool) {
	body := []byte(in)
	if chat {
		var err error
		if gjson.GetBytes(body, "stream").Bool() && (!gjson.GetBytes(body, "stream_options").Exists() || !gjson.GetBytes(body, "stream_options.include_usage").Exists()) {
			body, err = sjson.SetBytes(body, "stream_options.include_usage", true)
			if err != nil {
				return nil, false
			}
		}
		body, err = convertDeveloperRoleToSystem(body)
		if err != nil {
			return nil, false
		}
	}
	return body, true
}

func oaiDefaultModel(body []byte, mapping map[string]string) ([]byte, bool) {
	model := gjson.GetBytes(body, "model").String()
	body, err := sjson.SetBytes(body, "model", getMappedModel(model, mapping))
	return body, err == nil
}

func toMap(b []byte) (map[string]any, bool) {
	m, err := decodeMap(b)
	return m, err == nil
}

// officialQwenCompat：qwen.go TransformRequestBodyHeaders 的兼容分支
func officialQwenCompat(in string, mapping map[string]string, chat bool) (map[string]any, bool) {
	body, ok := oaiPre(in, chat)
	if !ok {
		return nil, false
	}
	modifiedBody := body
	model := gjson.GetBytes(modifiedBody, "model")
	mappedModel := ""
	if model.Exists() {
		rawModel := model.String()
		mappedModel = getMappedModel(rawModel, mapping)
		newBody, err := sjson.SetBytes(modifiedBody, "model", mappedModel)
		if err != nil {
			return nil, false
		}
		modifiedBody = newBody
	}
	if mappedModel != "" && requestBodyHasMessageReasoningContent(modifiedBody) && qwenSupportsPreserveThinking(mappedModel) {
		var err error
		modifiedBody, err = sjson.SetBytes(modifiedBody, "preserve_thinking", true)
		if err != nil {
			return nil, false
		}
	}
	return toMap(modifiedBody)
}

// officialZhipu：zhipuai.go TransformRequestBody（chat）
func officialZhipu(in string, mapping map[string]string, chat bool) (map[string]any, bool) {
	body, ok := oaiPre(in, chat)
	if !ok {
		return nil, false
	}
	if chat {
		reasoningEffort := gjson.GetBytes(body, "reasoning_effort").String()
		if reasoningEffort != "" {
			body, _ = sjson.SetBytes(body, "thinking", map[string]string{"type": "enabled"})
			body, _ = sjson.DeleteBytes(body, "reasoning_effort")
		}
		if requestBodyHasMessageReasoningContent(body) {
			body, _ = sjson.SetBytes(body, "thinking.clear_thinking", false)
		}
	}
	body, ok = oaiDefaultModel(body, mapping)
	if !ok {
		return nil, false
	}
	return toMap(body)
}

// officialOpenRouter：openrouter.go TransformRequestBody（chat，去掉 Claude ctx 分支）
func officialOpenRouter(in string, mapping map[string]string, chat bool) (map[string]any, bool) {
	body, ok := oaiPre(in, chat)
	if !ok {
		return nil, false
	}
	if chat {
		reasoningMaxTokens := gjson.GetBytes(body, "reasoning_max_tokens")
		if reasoningMaxTokens.Exists() && reasoningMaxTokens.Int() != 0 {
			modifiedBody, err := sjson.DeleteBytes(body, "reasoning_effort")
			if err != nil {
				modifiedBody = body
			}
			modifiedBody, err = sjson.SetBytes(modifiedBody, "reasoning.max_tokens", reasoningMaxTokens.Int())
			if err != nil {
				return nil, false
			}
			modifiedBody, err = sjson.DeleteBytes(modifiedBody, "reasoning_max_tokens")
			if err != nil {
				return nil, false
			}
			body = modifiedBody
		}
	}
	body, ok = oaiDefaultModel(body, mapping)
	if !ok {
		return nil, false
	}
	return toMap(body)
}

var vMapping = map[string]string{"m1": "qwen3.6-plus-x", "k": "kimi-k2.6", "gpt-*": "g", "*": "fallback"}

type variantCase struct {
	name     string
	official func(in string, mapping map[string]string, chat bool) (map[string]any, bool)
	stream   func(chat bool) *streamxform.Transformer
	inputs   []string
}

func variantCases() []variantCase {
	mk := func(v func() streamxform.OpenAIVariant, onlyIfPresent, detect bool) func(bool) *streamxform.Transformer {
		return func(chat bool) *streamxform.Transformer {
			var vv streamxform.OpenAIVariant
			if v != nil && chat {
				vv = v()
			}
			return streamxform.NewOpenAI(streamxform.OpenAIOptions{
				MapModel:           func(m string) string { return getMappedModel(m, vMapping) },
				ModelOnlyIfPresent: onlyIfPresent,
				DetectStream:       detect && chat,
				NormalizeUsage:     chat,
				CheckMessages:      chat,
				Variant:            vv,
			})
		}
	}
	return []variantCase{
		{"qwen兼容", officialQwenCompat, func(chat bool) *streamxform.Transformer {
			return streamxform.NewOpenAI(streamxform.OpenAIOptions{
				MapModel:           func(m string) string { return getMappedModel(m, vMapping) },
				ModelOnlyIfPresent: true,
				NormalizeUsage:     chat,
				CheckMessages:      chat,
				Variant:            &streamxform.QwenVariant{SupportsPreserveThinking: qwenSupportsPreserveThinking},
			})
		}, []string{
			`{"model":"m1","messages":[{"role":"user","content":"U"}]}`,
			`{"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","messages":[{"role":"assistant","content":"A","reasoning_content":"think"},{"role":"user","content":"U"}]}`,
			`{"model":"k","messages":[{"role":"assistant","reasoning_content":"","content":"A"},{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"zzz","messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"m1","preserve_thinking":false,"messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"m1","preserve_thinking":false,"messages":[{"role":"assistant","reasoning_content":""}]}`,
			`{"model":"m1","messages":[{"role":"assistant","reasoning_content":0},{"role":"user","content":"U"}]}`,
			`{"model":"m1","messages":[{"role":"assistant","reasoning_content":null}]}`,
			`{"model":"m1","messages":[{"role":"assistant","reasoning_content":{"a":1}}]}`,
			`{"model":"m1","messages":["s",1,null,{"reasoning_content":"x"}]}`,
			`{"model":"m1","stream":true,"messages":[{"role":"assistant","reasoning_content":"` + strings.Repeat("r", 5000) + `"}]}`,
			`{"messages":[{"role":"assistant","reasoning_content":"x"}],"model":"m1"}`,
		}},
		{"智谱", officialZhipu, mk(func() streamxform.OpenAIVariant { return &streamxform.ZhipuVariant{} }, false, true), []string{
			`{"model":"m1","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_effort":"high","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_effort":"","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_effort":"high","thinking":{"type":"disabled","x":1},"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","thinking":{"type":"disabled"},"messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"m1","messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"m1","reasoning_effort":"low","messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"model":"m1","reasoning_effort":5,"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_effort":null,"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","thinking":{"clear_thinking":true},"messages":[{"role":"assistant","reasoning_content":"x"}]}`,
			`{"messages":[{"role":"assistant","reasoning_content":"x"}],"thinking":{"type":"enabled"},"reasoning_effort":"medium","model":"gpt-4"}`,
		}},
		{"openrouter", officialOpenRouter, mk(func() streamxform.OpenAIVariant { return &streamxform.OpenRouterVariant{} }, false, true), []string{
			`{"model":"m1","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":1000,"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":0,"reasoning_effort":"high","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":1000,"reasoning_effort":"high","reasoning":{"exclude":true,"max_tokens":5},"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":"200","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":"2.5","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":12.9,"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":true,"messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_max_tokens":null,"reasoning_effort":"low","messages":[{"role":"user","content":"U"}]}`,
			`{"model":"m1","reasoning_effort":"low","reasoning":{"effort":"x"},"messages":[{"role":"user","content":"U"}]}`,
			`{"reasoning_effort":"low","messages":[{"role":"user","content":"U"}],"reasoning_max_tokens":-5,"model":"m1"}`,
		}},
	}
}

func TestVariantsDifferential(t *testing.T) {
	for _, vc := range variantCases() {
		fmt.Printf("  == %s\n", vc.name)
		for _, in := range vc.inputs {
			for _, chat := range []bool{true, false} {
				off, ok := vc.official(in, vMapping, chat)
				if !ok {
					if chat {
						fmt.Printf("  %-52s 官方 struct 校验失败\n", trunc(in, 52))
					}
					continue
				}
				for _, cs := range []int{1, 5, 4096} {
					str, sok, why := runStream(vc.stream(chat), in, cs)
					if !sok {
						if chat && hasDeveloper(in) {
							continue
						}
						t.Errorf("%s chunk=%d chat=%v 意外回落: %s\n  %s", vc.name, cs, chat, why, in)
						continue
					}
					if d := diffMaps(off, str); d != "" {
						t.Errorf("%s chunk=%d chat=%v 不一致: %s\n  输入: %s", vc.name, cs, chat, d, in)
					} else if cs == 4096 && chat {
						fmt.Printf("  %-52s ✓ 一致\n", trunc(in, 52))
					}
				}
			}
		}
	}
}

// genVariantRequest 在 genRequest 基础上加入各 variant 关心的字段
func genVariantRequest(r *rand.Rand) string {
	in := genRequest(r)
	pick := func(xs ...string) string { return xs[r.Intn(len(xs))] }
	extra := []string{}
	if r.Intn(3) == 0 {
		extra = append(extra, `"preserve_thinking":`+pick("true", "false", "null", `"x"`))
	}
	if r.Intn(3) == 0 {
		extra = append(extra, `"thinking":`+pick(`{"type":"disabled"}`, `{"type":"enabled","clear_thinking":true}`, `{}`, `null`, `"str"`, `[1]`))
	}
	if r.Intn(3) == 0 {
		extra = append(extra, `"reasoning":`+pick(`{"effort":"low"}`, `{"max_tokens":5,"exclude":true}`, `{}`, `null`, `7`))
	}
	if r.Intn(3) == 0 {
		extra = append(extra, `"reasoning_max_tokens":`+pick("0", "100", "-3", "2.9", `"15"`, `"x"`, "true", "null", `""`))
	}
	if r.Intn(2) == 0 {
		in = strings.Replace(in, `"model":`, `"model_x":`, 1) // 有时去掉 model
		in = strings.Replace(in, `"model_x":`, `"model":`, 0)
	}
	if len(extra) > 0 {
		in = "{" + strings.Join(extra, ",") + "," + in[1:]
	}
	// 给部分 message 加 reasoning_content 的各种形态
	if r.Intn(2) == 0 {
		in = strings.Replace(in, `"role":"assistant"`, `"role":"assistant","reasoning_content":`+pick(`"t"`, `""`, `null`, `0`, `1`, `{"a":1}`, `[]`, `"`+strings.Repeat("z", 3000)+`"`), 1)
	}
	return in
}

func TestVariantsFuzz(t *testing.T) {
	seed := int64(time.Now().UnixNano())
	r := rand.New(rand.NewSource(seed))
	N := fuzzN(6000)
	for _, vc := range variantCases() {
		same, offFail, fb := 0, 0, 0
		for i := 0; i < N; i++ {
			in := genVariantRequest(r)
			chat := r.Intn(4) != 0
			off, ok := vc.official(in, vMapping, chat)
			if !ok {
				offFail++
				continue
			}
			chunk := []int{1, 3, 17, 64, 4096}[r.Intn(5)]
			str, sok, why := runStream(vc.stream(chat), in, chunk)
			if !sok {
				if chat && hasDeveloper(in) || strings.Contains(why, "不是对象") || strings.Contains(why, "messages 不是数组") || strings.Contains(why, "重复 key") {
					fb++
					continue
				}
				t.Fatalf("%s 第 %d 例意外回落 (chunk=%d chat=%v): %s\n  输入: %s", vc.name, i, chunk, chat, why, in)
			}
			if d := diffMaps(off, str); d != "" {
				t.Fatalf("%s 第 %d 例不一致 (chunk=%d chat=%v)\n  输入: %s\n  差异: %s", vc.name, i, chunk, chat, in, d)
			}
			same++
		}
		fmt.Printf("  %s 随机 %d 例 (seed=%d): 一致 %d, 官方失败 %d, 已知回落 %d, 不一致 0\n", vc.name, N, seed, same, offFail, fb)
	}
}

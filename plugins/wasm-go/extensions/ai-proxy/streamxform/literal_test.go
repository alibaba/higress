package streamxform

import (
	"encoding/json"
	"strings"
	"testing"
)

// feedAll 分块喂入并收尾，返回 (输出, 是否放行, 不支持原因)。
func feedAll(tr *Transformer, in string, chunk int) (string, bool, string) {
	var sb strings.Builder
	for i := 0; i < len(in); i += chunk {
		j := i + chunk
		if j > len(in) {
			j = len(in)
		}
		tr.Write([]byte(in[i:j]))
		sb.Write(tr.Out())
	}
	sb.Write(tr.Finish())
	if u, why := tr.Unsupported(); u {
		return "", false, why
	}
	return sb.String(), true, ""
}

// 扫描器对标量字面量 / 字符串转义 / 空白按 JSON 文法严格校验：
// encoding/json 会拒绝的输入，流式必须判定不支持（回落后由官方路径返回 400），
// 无论它落在派发帧、Skip 区域还是 Pass 区域。
func TestStrictLiterals(t *testing.T) {
	bad := map[string]string{
		"nul 顶层":      `{"model":"m","stream":nul,"messages":[{"role":"user","content":"U"}]}`,
		"tru 顶层":      `{"model":"m","stream":tru,"messages":[{"role":"user","content":"U"}]}`,
		"truee":       `{"model":"m","stream":truee,"messages":[{"role":"user","content":"U"}]}`,
		"前导0":         `{"model":"m","max_tokens":01,"messages":[{"role":"user","content":"U"}]}`,
		"1.":          `{"model":"m","temperature":1.,"messages":[{"role":"user","content":"U"}]}`,
		"1e":          `{"model":"m","temperature":1e,"messages":[{"role":"user","content":"U"}]}`,
		"1e+":         `{"model":"m","temperature":1e+,"messages":[{"role":"user","content":"U"}]}`,
		"-":           `{"model":"m","temperature":-,"messages":[{"role":"user","content":"U"}]}`,
		"+1":          `{"model":"m","temperature":+1,"messages":[{"role":"user","content":"U"}]}`,
		".5":          `{"model":"m","temperature":.5,"messages":[{"role":"user","content":"U"}]}`,
		"NaN":         `{"model":"m","temperature":NaN,"messages":[{"role":"user","content":"U"}]}`,
		"Skip区域里":     `{"model":"m","zzz":{"a":[nul]},"messages":[{"role":"user","content":"U"}]}`,
		"Skip区域里数字":   `{"model":"m","zzz":{"a":1.e5},"messages":[{"role":"user","content":"U"}]}`,
		"Pass区域里":     `{"model":"m","messages":[{"role":"user","content":"U","zzz":[+1]}]}`,
		"Pass区域里字面量":  `{"model":"m","messages":[{"role":"user","content":"U","zzz":{"a":fals}}]}`,
		"末尾字面量":       `{"model":"m","messages":[{"role":"user","content":"U"}],"zzz":tr}`,
		"非法转义":        `{"model":"m","messages":[{"role":"user","content":"a\x"}]}`,
		"u转义不足":       `{"model":"m","messages":[{"role":"user","content":"a\u12G4"}]}`,
		"u转义被引号截断":    `{"model":"m","messages":[{"role":"user","content":"a\u123"}]}`,
		"控制字符":        "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"a\x01b\"}]}",
		"控制字符 Pass区域": "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"U\",\"zzz\":\"a\nb\"}]}",
		"key非法转义":     `{"model":"m","messages":[{"role":"user","content":"U","z\q":1}]}`,
		"key控制字符":     "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"U\",\"z\x02\":1}]}",
		"非JSON空白 VT":  "{\"model\":\"m\",\x0b\"messages\":[{\"role\":\"user\",\"content\":\"U\"}]}",
		"非JSON空白 FF":  "{\"model\":\"m\",\"messages\":[\x0c{\"role\":\"user\",\"content\":\"U\"}]}",
		"非JSON空白 NUL": "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"U\"}]\x00}",
	}
	good := map[string]string{
		"数字边界":   `{"model":"m","temperature":-0,"top_p":0.5E+1,"max_tokens":100,"zzz":{"a":-0.0e-0,"b":[1E2,0,-1,12.5e-7]},"messages":[{"role":"user","content":"U"}]}`,
		"转义边界":   `{"model":"m","messages":[{"role":"user","content":"\u00e9\/\"\\\ud83d\ude00\b\f\n\r\t"}]}`,
		"key转义":  `{"model":"m","messages":[{"role":"user","content":"U","z\u0041\"":1}]}`,
		"JSON空白": "{\t\"model\"\r:\n\"m\" ,\"messages\":[ {\"role\":\"user\",\"content\":\"U\"} ]\n}\n",
	}
	mk := map[string]func() *Transformer{
		"claude": func() *Transformer { return NewClaude(ClaudeOptions{}) },
		"openai": func() *Transformer { return NewOpenAI(OpenAIOptions{}) },
	}
	for pn, newT := range mk {
		for name, in := range bad {
			if err := json.Unmarshal([]byte(in), new(map[string]any)); err == nil {
				t.Fatalf("用例 %q 本应是非法 JSON", name)
			}
			for _, cs := range []int{1, 2, 3, 7, 4096} {
				if out, ok, _ := feedAll(newT(), in, cs); ok {
					t.Errorf("[%s] %s chunk=%d: 非法输入被放行: %s", pn, name, cs, out)
				}
			}
		}
		for name, in := range good {
			if err := json.Unmarshal([]byte(in), new(map[string]any)); err != nil {
				t.Fatalf("用例 %q 本应是合法 JSON: %v", name, err)
			}
			for _, cs := range []int{1, 2, 3, 7, 4096} {
				out, ok, why := feedAll(newT(), in, cs)
				if !ok {
					t.Errorf("[%s] %s chunk=%d: 合法输入被拒: %s", pn, name, cs, why)
					continue
				}
				if !json.Valid([]byte(out)) {
					t.Errorf("[%s] %s chunk=%d: 输出不是合法 JSON: %s", pn, name, cs, out)
				}
			}
		}
	}
}

// 标量子类型：null / bool / number 在 OnStart 里可区分，协议据此做"null 视为缺失、其它类型错误回落"。
func TestScalarKinds(t *testing.T) {
	cases := []struct {
		in   string
		want ValueKind
	}{
		{`null`, KindNull}, {`true`, KindBool}, {`false`, KindBool},
		{`0`, KindNumber}, {`-1.5e3`, KindNumber}, {`"s"`, KindString}, {`{}`, KindObject}, {`[]`, KindArray},
	}
	for _, c := range cases {
		var got ValueKind
		p := &kindProbe{on: func(k ValueKind) { got = k }}
		tr := NewTransformer(p)
		_, ok, why := feedAll(tr, `{"k":`+c.in+`}`, 1)
		if !ok {
			t.Fatalf("%s: %s", c.in, why)
		}
		if got != c.want {
			t.Errorf("%s: kind=%v want %v", c.in, got, c.want)
		}
	}
}

type kindProbe struct {
	BaseProtocol
	on func(ValueKind)
}

func (p *kindProbe) OnKey(t *Transformer) Action { return Probe() }
func (p *kindProbe) OnStart(t *Transformer, kind ValueKind) Action {
	p.on(kind)
	return Pass()
}

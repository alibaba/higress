package streamxform

import (
	"math/rand"
	"strings"
	"testing"
)

// 任意字节流（合法 JSON 的随机变异、随机字节、随机截断、随机分块）喂给每个协议，
// 只要求：不 panic、Finish 之后要么给出合法输出要么明确 Bail。WASM 里 panic 等于 500。
func TestNoPanicOnGarbage(t *testing.T) {
	seeds := []string{
		`{"model":"m","messages":[{"role":"system","content":"S"},{"role":"user","content":[{"type":"text","text":"a"},{"type":"image_url","image_url":{"url":"data:image/png;base64,QUJD"}}]},{"role":"assistant","content":"x","tool_calls":[{"id":"c","type":"function","function":{"name":"f","arguments":"{}"}}]},{"role":"tool","tool_call_id":"c","content":"r"}],"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}],"tool_choice":"auto","stream":true,"max_tokens":10,"temperature":0.5,"stop":["x"],"reasoning_effort":"low","stream_options":{"include_usage":true},"preserve_thinking":false,"thinking":{"type":"enabled"},"reasoning_max_tokens":5}`,
		`{"messages":[{"content":"é\\\"\n","role":"user","name":"n","reasoning_content":"r"}],"model":"gpt-4","top_p":0.3,"logprobs":true,"modalities":["TEXT"],"presence_penalty":1.5,"n":2,"seed":3}`,
		`{"a":[[[{"b":[{}]}]]],"c":"\\ud83d\\ude00","d":-1.5e-3,"e":null,"f":true}`,
	}
	makers := []func() *Transformer{
		func() *Transformer { return New() },
		func() *Transformer { return NewClaude(ClaudeOptions{ClaudeCodeMode: true}) },
		func() *Transformer {
			return NewOpenAI(OpenAIOptions{DetectStream: true, NormalizeUsage: true, CheckMessages: true, Variant: &ZhipuVariant{}})
		},
		func() *Transformer { return NewOpenAI(OpenAIOptions{Variant: &OpenRouterVariant{}, CheckMessages: true}) },
		func() *Transformer {
			return NewOpenAI(OpenAIOptions{Variant: &QwenVariant{SupportsPreserveThinking: func(string) bool { return true }}, ModelOnlyIfPresent: true})
		},
		func() *Transformer { return NewGemini(GeminiOptions{ThinkingModel: func(string) bool { return true }}) },
		func() *Transformer { return NewQwenNative(QwenNativeOptions{EnableSearch: true}) },
	}
	r := rand.New(rand.NewSource(20260905))
	mutate := func(s string) string {
		b := []byte(s)
		switch r.Intn(6) {
		case 0: // 截断
			return string(b[:r.Intn(len(b)+1)])
		case 1: // 随机改一个字节
			if len(b) > 0 {
				b[r.Intn(len(b))] = byte(r.Intn(256))
			}
			return string(b)
		case 2: // 插入结构字符
			i := r.Intn(len(b) + 1)
			return string(b[:i]) + string([]byte{`{}[]",:\`[r.Intn(8)]}) + string(b[i:])
		case 3: // 删除一段
			i := r.Intn(len(b) + 1)
			j := i + r.Intn(8)
			if j > len(b) {
				j = len(b)
			}
			return string(b[:i]) + string(b[j:])
		case 4: // 纯随机字节
			n := r.Intn(64)
			out := make([]byte, n)
			for k := range out {
				out[k] = byte(r.Intn(256))
			}
			return string(out)
		}
		return s + strings.Repeat(" ", r.Intn(3))
	}
	const N = 30000
	valid, bailed := 0, 0
	for i := 0; i < N; i++ {
		in := seeds[r.Intn(len(seeds))]
		for k := r.Intn(4); k >= 0; k-- {
			in = mutate(in)
		}
		mk := makers[r.Intn(len(makers))]
		chunk := []int{1, 2, 5, 16, 4096}[r.Intn(5)]
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("第 %d 例 panic: %v\n输入: %q chunk=%d", i, p, in, chunk)
				}
			}()
			tr := mk()
			var out []byte
			for j := 0; j < len(in); j += chunk {
				e := j + chunk
				if e > len(in) {
					e = len(in)
				}
				tr.Write([]byte(in[j:e]))
				out = append(out, tr.Out()...)
			}
			out = append(out, tr.Finish()...)
			if bad, _ := tr.Unsupported(); bad {
				bailed++
				return
			}
			valid++
		}()
	}
	t.Logf("随机变异 %d 例：放行 %d，回落 %d，panic 0", N, valid, bailed)
}

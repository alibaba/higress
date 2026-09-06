package streamxform

import (
	"strings"
	"testing"
)

// 吞吐基准：长字符串直通、结构密集（大量小 kv 与标量）、多模态 base64。
// 用于确认逐字节语法校验没有让扫描器变慢。
func benchBody(kind string) []byte {
	switch kind {
	case "longstr":
		return []byte(`{"model":"m","messages":[{"role":"user","content":"` + strings.Repeat("abcdefgh", 128<<10) + `"}],"stream":true}`)
	case "dense":
		var sb strings.Builder
		sb.WriteString(`{"model":"m","messages":[{"role":"user","content":"U"}],"tools":[`)
		for i := 0; i < 2000; i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(`{"type":"function","function":{"name":"f","description":"d","parameters":{"type":"object","properties":{"a":{"type":"number","enum":[1,2.5,-3e-2,true,false,null]},"b":{"type":"string","description":"x\nyé"}},"required":["a","b"]}}}`)
		}
		sb.WriteString(`],"stream":true}`)
		return []byte(sb.String())
	default: // base64 图片
		return []byte(`{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"看图"},{"type":"image_url","image_url":{"url":"data:image/png;base64,` + strings.Repeat("iVBORw0KGgo=", 96<<10) + `"}}]}],"stream":true}`)
	}
}

func benchRun(b *testing.B, kind string, mk func() *Transformer) {
	body := benchBody(kind)
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr := mk()
		for off := 0; off < len(body); off += 4096 {
			end := off + 4096
			if end > len(body) {
				end = len(body)
			}
			tr.Write(body[off:end])
			tr.Out()
		}
		tr.Finish()
		if u, why := tr.Unsupported(); u {
			b.Fatal(why)
		}
	}
}

func BenchmarkClaudeLongString(b *testing.B) {
	benchRun(b, "longstr", func() *Transformer { return NewClaude(ClaudeOptions{}) })
}
func BenchmarkClaudeDense(b *testing.B) {
	benchRun(b, "dense", func() *Transformer { return NewClaude(ClaudeOptions{}) })
}
func BenchmarkClaudeImage(b *testing.B) {
	benchRun(b, "image", func() *Transformer { return NewClaude(ClaudeOptions{}) })
}
func BenchmarkPassthroughLongString(b *testing.B) {
	benchRun(b, "longstr", func() *Transformer { return NewOpenAI(OpenAIOptions{}) })
}
func BenchmarkPassthroughDense(b *testing.B) {
	benchRun(b, "dense", func() *Transformer { return NewOpenAI(OpenAIOptions{}) })
}

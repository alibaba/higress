package provider

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/streamxform"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// officialQwenNative：qwen.go onChatCompletionRequestBody 的纯函数部分
// （parseRequestAndMapModel 的 decode + 空 model 校验 + 映射 → buildQwenTextGenerationRequest）。
func officialQwenNativeErr(in string, mapping map[string]string, enableSearch bool) (map[string]any, error) {
	req := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest([]byte(in), req); err != nil {
		return nil, err
	}
	if req.Model == "" {
		return nil, fmt.Errorf("missing model in request")
	}
	req.Model = getMappedModel(req.Model, mapping)
	m := &qwenProvider{config: ProviderConfig{qwenEnableSearch: enableSearch, modelMapping: mapping}}
	body, err := m.buildQwenTextGenerationRequest(qwenCtxStub{}, req, req.Stream)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// qwenCtxStub 只需要 SetContext（官方在 streaming 时写 incrementalStreaming 上下文键）
type qwenCtxStub struct{ wrapper.HttpContext }

func (qwenCtxStub) SetContext(string, interface{}) {}

var qwenNativeMapping = map[string]string{"m1": "qwen3.6-plus-x", "k": "kimi-k2.6", "vl": "qwen-vl-max", "gpt-*": "g", "*": "fallback"}

func newQwenNativeStream(enableSearch bool) *streamxform.Transformer {
	return streamxform.NewQwenNative(streamxform.QwenNativeOptions{
		MapModel: func(m string) (string, error) {
			if m == "" {
				return "", fmt.Errorf("missing model in request")
			}
			return getMappedModel(m, qwenNativeMapping), nil
		},
		SupportsPreserveThinking: qwenSupportsPreserveThinking,
		EnableSearch:             enableSearch,
	})
}

var qwenNativeCases = []string{
	`{"model":"m1","messages":[{"role":"user","content":"U"}]}`,
	`{"model":"x","messages":[{"role":"system","content":"S"},{"role":"user","content":"U"},{"role":"assistant","content":"A"}]}`,
	`{"model":"m1","stream":true,"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","stream":true,"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}],"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","stream":true,"tools":[],"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","stream":true,"tools":null,"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","max_tokens":100,"n":2,"seed":7,"temperature":0.5,"top_p":0.9,"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","top_p":0,"temperature":0,"max_tokens":0,"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","top_p":1.5,"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","messages":[{"role":"assistant","content":"A","reasoning_content":"think"},{"role":"user","content":"U"}]}`,
	`{"model":"k","messages":[{"role":"assistant","reasoning_content":"","content":"A"},{"role":"assistant","reasoning_content":"x"}]}`,
	`{"model":"m1","messages":[{"role":"user","content":"U","name":"n"},{"role":"user","content":"U","name":""}]}`,
	`{"model":"m1","messages":[{"role":"assistant","content":"x","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]}]}`,
	`{"model":"m1","messages":[{"role":"assistant","content":"x","tool_calls":[]},{"role":"assistant","content":"y","tool_calls":null}]}`,
	`{"model":"m1","messages":[{"role":"tool","tool_call_id":"c1","content":"r"}]}`,
	`{"model":"m1","messages":[{"content":"U","role":"user"},{}]}`,
	`{"model":"m1","messages":[{"role":"user"}]}`,
	`{"model":"m1","messages":[{"role":"user","content":null}]}`,
	`{"model":"m1","messages":[{"role":"user","content":{"a":1}}]}`,
	`{"model":"m1","messages":[{"role":"user","content":[]}]}`,
	`{"model":"m1","messages":[{"role":"user","content":[{"type":"text","text":"a"},{"type":"text","text":""},{"text":"b","type":"text"}]}]}`,
	`{"model":"vl","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA","detail":"high"}},{"type":"image_url","image_url":{"url":"https://x/y.png"}},{"type":"image_url","image_url":{"url":""}}]}]}`,
	`{"model":"vl","messages":[{"role":"user","content":[{"image_url":{"url":"https://x/y.png"},"type":"image_url"}]}]}`,
	`{"model":"m1","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"AA","format":"wav"}},{"type":"file","file":{"file_id":"f"}},{"type":"video","x":1},{"text":"notype"},"s",1,null]}]}`,
	`{"model":"m1","messages":[{"role":"user","content":[{"type":"video","text":"x"}]}]}`,
	`{"model":"m1","messages":[{"role":"user","content":"` + strings.Repeat("y", 100000) + `"}],"stream":true}`,
	`{"model":"m1","stop":["E"],"presence_penalty":1,"tool_choice":"auto","stream_options":{"include_usage":true},"messages":[{"role":"user","content":"U"}]}`,
	`{"messages":[{"role":"user","content":"U"}]}`,
	`{"model":"m1","messages":[]}`,
}

func TestQwenNativeDifferential(t *testing.T) {
	for _, in := range qwenNativeCases {
		for _, search := range []bool{false, true} {
			off, oerr := officialQwenNativeErr(in, qwenNativeMapping, search)
			for _, cs := range []int{1, 7, 4096} {
				str, sok, why := runStream(newQwenNativeStream(search), in, cs)
				if oerr != nil {
					if sok {
						t.Errorf("官方失败但流式放行: %s (%v)", trunc(in, 80), oerr)
					} else if cs == 4096 && !search {
						fmt.Printf("  %-60s 官方失败，流式回落 ✓ (%s)\n", trunc(in, 60), why)
					}
					continue
				}
				if !sok {
					t.Errorf("chunk=%d 意外回落: %s\n  %s", cs, why, in)
					continue
				}
				if d := diffMaps(off, str); d != "" {
					t.Errorf("chunk=%d search=%v 不一致: %s\n  输入: %s", cs, search, d, trunc(in, 200))
				} else if cs == 4096 && !search {
					fmt.Printf("  %-60s ✓ 一致\n", trunc(in, 60))
				}
			}
		}
	}
}

func TestQwenNativeFuzz(t *testing.T) {
	seed := int64(time.Now().UnixNano())
	r := rand.New(rand.NewSource(seed))
	N := fuzzN(60000)
	same, offFail, lenient, fb := 0, 0, 0, 0
	for i := 0; i < N; i++ {
		in := genRequest(r)
		if r.Intn(2) == 0 {
			in = strings.Replace(in, `"role":"assistant"`, `"role":"assistant","reasoning_content":`+[]string{`"t"`, `""`, `"` + strings.Repeat("z", 3000) + `"`}[r.Intn(3)], 1)
		}
		search := r.Intn(2) == 0
		off, oerr := officialQwenNativeErr(in, qwenNativeMapping, search)
		chunk := []int{1, 3, 17, 64, 4096}[r.Intn(5)]
		str, sok, why := runStream(newQwenNativeStream(search), in, chunk)
		if oerr != nil {
			offFail++
			if sok {
				if strings.Contains(oerr.Error(), "cannot unmarshal") {
					lenient++
					continue
				}
				t.Fatalf("第 %d 例官方失败但流式放行 (chunk=%d): %v\n  输入: %s", i, chunk, oerr, in)
			}
			continue
		}
		if !sok {
			if strings.Contains(why, "重复的 key") || strings.Contains(why, "panic") {
				fb++
				continue
			}
			t.Fatalf("第 %d 例意外回落 (chunk=%d): %s\n  输入: %s", i, chunk, why, in)
		}
		if d := diffMaps(off, str); d != "" {
			t.Fatalf("第 %d 例不一致 (chunk=%d)\n  输入: %s\n  差异: %s", i, chunk, in, d)
		}
		same++
	}
	fmt.Printf("  Qwen 原生随机 %d 例 (seed=%d): 一致 %d, 官方失败 %d (其中丢弃字段类型错误、流式放行 %d), 已知回落 %d, 不一致 0\n", N, seed, same, offFail, lenient, fb)
}

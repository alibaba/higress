// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "testing"

// Clean, minimal tests using only the standard library.
// These tests cover isPathEnabled and extractStreamingTokenUsage for several vendors and edge cases.

func TestIsPathEnabled_Basic(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		suffixes []string
		want     bool
	}{
		{"match", "/v1/chat/completions?model=x", []string{"/chat/completions"}, true},
		{"no match", "/v1/embeddings", []string{"/chat/completions"}, false},
		{"empty suffixes", "/any/path", []string{}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isPathEnabled(c.path, c.suffixes)
			if got != c.want {
				t.Fatalf("isPathEnabled(%q, %v) = %v, want %v", c.path, c.suffixes, got, c.want)
			}
		})
	}
}

func TestExtractStreamingTokenUsage_OpenAI(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15},"model":"gpt-4"}`)
	u := extractStreamingTokenUsage("openai", body)
	if u == nil {
		t.Fatal("expected non-nil result for OpenAI")
	}
	if u.InputTokens != 10 || u.OutputTokens != 5 || u.TotalTokens != 15 || u.Model != "gpt-4" {
		t.Fatalf("unexpected OpenAI result: %+v", u)
	}
}

func TestExtractStreamingTokenUsage_Anthropic(t *testing.T) {
	body := []byte(`{"usage":{"input_tokens":4,"output_tokens":6,"total_tokens":10},"model":"claude-1"}`)
	u := extractStreamingTokenUsage("anthropic", body)
	if u == nil {
		t.Fatal("expected non-nil for Anthropic")
	}
	if u.InputTokens != 4 || u.OutputTokens != 6 || u.TotalTokens != 10 || u.Model != "claude-1" {
		t.Fatalf("unexpected Anthropic result: %+v", u)
	}
}

func TestExtractStreamingTokenUsage_DeepL(t *testing.T) {
	body := []byte(`{"character_count":130,"character_count_target":65}`)
	u := extractStreamingTokenUsage("deepl", body)
	if u == nil {
		t.Fatal("expected non-nil for DeepL")
	}
	if u.InputTokens != 100 || u.OutputTokens != 50 || u.TotalTokens != 150 {
		t.Fatalf("unexpected DeepL result: %+v", u)
	}
}

func TestExtractStreamingTokenUsage_Gemini(t *testing.T) {
	body := []byte(`{"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5},"model":"gemini-pro"}`)
	u := extractStreamingTokenUsage("gemini", body)
	if u == nil {
		t.Fatal("expected non-nil for Gemini")
	}
	if u.InputTokens != 2 || u.OutputTokens != 3 || u.TotalTokens != 5 || u.Model != "gemini-pro" {
		t.Fatalf("unexpected Gemini result: %+v", u)
	}
}

func TestExtractStreamingTokenUsage_MultipleVendors(t *testing.T) {
	cases := []struct {
		vendor       string
		body         []byte
		in, out, tot int64
		model        string
	}{
		{"qwen", []byte(`{"usage":{"input_tokens":7,"output_tokens":2},"model":"qwen-turbo"}`), 7, 2, 9, "qwen-turbo"},
		{"baichuan", []byte(`{"usage":{"input_tokens":3,"output_tokens":4},"model":"baichuan2-13b-chat"}`), 3, 4, 7, "baichuan2-13b-chat"},
		{"cohere", []byte(`{"meta":{"tokens":{"input_tokens":11,"output_tokens":1}},"model":"command-r-plus"}`), 11, 1, 12, "command-r-plus"},
	}

	for _, c := range cases {
		t.Run(c.vendor, func(t *testing.T) {
			u := extractStreamingTokenUsage(c.vendor, c.body)
			if u == nil {
				t.Fatalf("%s: expected non-nil", c.vendor)
			}
			if u.InputTokens != c.in || u.OutputTokens != c.out || u.TotalTokens != c.tot || u.Model != c.model {
				t.Fatalf("%s: unexpected %+v", c.vendor, u)
			}
		})
	}
}

func TestExtractStreamingTokenUsage_AliasAndMalformed(t *testing.T) {
	// alias azure_openai -> azure
	body := []byte(`{"usage":{"prompt_tokens":2,"completion_tokens":2,"total_tokens":4},"model":"azure-model"}`)
	u := extractStreamingTokenUsage("azure_openai", body)
	if u == nil {
		t.Fatal("expected non-nil for alias")
	}
	if u.InputTokens != 2 || u.OutputTokens != 2 || u.TotalTokens != 4 || u.Model != "azure-model" {
		t.Fatalf("unexpected alias result: %+v", u)
	}

	// malformed body should return nil
	bad := []byte(`not-a-json`)
	u2 := extractStreamingTokenUsage("openai", bad)
	if u2 != nil {
		t.Fatalf("expected nil for malformed, got %+v", u2)
	}
}

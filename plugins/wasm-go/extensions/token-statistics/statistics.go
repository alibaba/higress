package main

import (
	"github.com/tidwall/gjson"
)

type extractTokenUsage interface {
	ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage
}

type OpenAI struct{}
type AzureOpenAI struct{}
type Anthropic struct{}
type GoogleGemini struct{}
type DeepSeek struct{}
type Qwen struct{}

// - mistral：https://docs.mistral.ai/guides/tokenization/
//   - gemini：https://ai.google.dev/gemini-api/docs/tokens?hl=zh-cn&lang=go
//   - claude code：https://docs.anthropic.com/zh-TW/docs/build-with-claude/token-counting
//   - 腾讯混元：https://cloud.tencent.com/document/product/1729/101835
//   - 字节豆包：https://www.volcengine.com/docs/82379/1528728
//   - 智谱：https://docs.bigmodel.cn/api-reference/%E6%A8%A1%E5%9E%8B-api/%E6%96%87%E6%9C%AC%E5%88%86%E8%AF%8D%E5%99%A8

// 从OpenAI格式响应中提取Token使用量
func (openai *OpenAI) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "usage"); usageData.Exists() {
		usage.InputTokens = usageData.Get("prompt_tokens").Int()
		usage.OutputTokens = usageData.Get("completion_tokens").Int()
		usage.TotalTokens = usageData.Get("total_tokens").Int()
		usage.Model = gjson.GetBytes(body, "model").String()
		return usage
	}

	return nil
}

func (azureOpenAI *AzureOpenAI) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "usage"); usageData.Exists() {
		usage.InputTokens = usageData.Get("prompt_tokens").Int()
		usage.OutputTokens = usageData.Get("completion_tokens").Int()
		usage.TotalTokens = usageData.Get("total_tokens").Int()
		usage.Model = gjson.GetBytes(body, "model").String()
		return usage
	}

	return nil
}

// 从Anthropic格式响应中提取Token使用量
func (anthropic *Anthropic) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "usage"); usageData.Exists() {
		usage.InputTokens = usageData.Get("input_tokens").Int()
		usage.OutputTokens = usageData.Get("output_tokens").Int()
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
		usage.Model = gjson.GetBytes(body, "model").String()
		return usage
	}

	return nil
}

// 从Google Gemini格式响应中提取Token使用量
func (gemini *GoogleGemini) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "usageMetadata"); usageData.Exists() {
		usage.InputTokens = usageData.Get("promptTokenCount").Int()
		usage.OutputTokens = usageData.Get("candidatesTokenCount").Int()
		usage.TotalTokens = usageData.Get("totalTokenCount").Int()
		// Gemini没有直接返回模型名称，需要从请求中获取
		return usage
	}

	return nil
}

func (qwen *Qwen) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "usage"); usageData.Exists() {
		usage.InputTokens = usageData.Get("input_tokens").Int()
		usage.OutputTokens = usageData.Get("output_tokens").Int()
		usage.TotalTokens = usageData.Get("total_tokens").Int()
		usage.Model = gjson.GetBytes(body, "model").String()
		return usage
	}

	return nil
}

func (deepseek *DeepSeek) ExtractTokenUsage(json gjson.Result, body []byte) *TokenUsage {
	usage := &TokenUsage{}

	if usageData := gjson.GetBytes(body, "token_usage"); usageData.Exists() {
		usage.InputTokens = usageData.Get("input_tokens").Int()
		usage.OutputTokens = usageData.Get("output_tokens").Int()
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
		usage.Model = gjson.GetBytes(body, "model").String()
		return usage
	}

	return nil
}

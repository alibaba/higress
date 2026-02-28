// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-a2as",
		wrapper.ParseConfig(ParseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
	)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config A2ASConfig) types.Action {
	ctx.DisableReroute()
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, globalConfig A2ASConfig, body []byte) types.Action {
	consumer, err := proxywasm.GetHttpRequestHeader("X-Mse-Consumer")
	if err == nil && consumer != "" {
		log.Debugf("[A2AS] Request from consumer: %s", consumer)
	}

	config := globalConfig.MergeConsumerConfig(consumer)

	if !isChatCompletionRequest(body) {
		log.Debugf("[A2AS] Not a chat completion request, skipping A2AS processing")
		return types.ActionContinue
	}

	// 签名验证（如果启用）
	if config.AuthenticatedPrompts.Enabled {
		verifiedBody, err := verifyAndRemoveEmbeddedHashes(config.AuthenticatedPrompts, body)
		if err != nil {
			log.Errorf("[A2AS] Signature verification failed: %v", err)
			_ = proxywasm.SendHttpResponse(403, [][2]string{
				{"content-type", "application/json"},
			}, []byte(`{"error":"unauthorized","message":"Invalid or missing prompt signature"}`), -1)
			return types.ActionPause
		}
		body = verifiedBody
		log.Debugf("[A2AS] Signature verification passed and hashes removed")
	}

	modifiedBody, err := applyA2ASTransformations(config, body)
	if err != nil {
		log.Errorf("[A2AS] Failed to apply transformations: %v", err)
		_ = proxywasm.SendHttpResponse(500, [][2]string{{"content-type", "application/json"}},
			[]byte(`{"error":"internal_error","message":"A2AS transformation failed"}`), -1)
		return types.ActionPause
	}

	if config.BehaviorCertificates.Enabled {
		if denied, tool := checkToolPermissions(config.BehaviorCertificates, modifiedBody); denied {
			log.Warnf("[A2AS] Tool call denied by behavior certificate: %s", tool)
			_ = proxywasm.SendHttpResponse(403, [][2]string{
				{"content-type", "application/json"},
			}, []byte(`{"error":"forbidden","message":"`+config.BehaviorCertificates.DenyMessage+`","denied_tool":"`+tool+`"}`), -1)
			return types.ActionPause
		}
	}

	if err := proxywasm.ReplaceHttpRequestBody(modifiedBody); err != nil {
		log.Errorf("[A2AS] Failed to replace request body: %v", err)
		_ = proxywasm.SendHttpResponse(500, [][2]string{
			{"content-type", "application/json"},
		}, []byte(`{"error":"internal_error","message":"Failed to apply security transformations"}`), -1)
		return types.ActionPause
	}

	log.Debugf("[A2AS] Successfully applied A2AS transformations")
	return types.ActionContinue
}

func isChatCompletionRequest(body []byte) bool {
	messages := gjson.GetBytes(body, "messages")
	return messages.Exists() && messages.IsArray()
}

func applyA2ASTransformations(config A2ASConfig, body []byte) ([]byte, error) {
	rawMessages := gjson.GetBytes(body, "messages")
	if !rawMessages.Exists() {
		return body, nil
	}

	newMessages := make([]map[string]interface{}, 0)

	// 注入 In-Context Defenses 作为系统消息
	if config.InContextDefenses.Enabled && config.InContextDefenses.Position == "as_system" {
		defenseContent := BuildDefenseBlock(config.InContextDefenses.Template)
		if config.InContextDefenses.Template == "custom" && config.InContextDefenses.CustomPrompt != "" {
			defenseContent = config.InContextDefenses.CustomPrompt
		}
		defenseMsg := map[string]interface{}{
			"role":    "system",
			"content": defenseContent,
		}
		newMessages = append(newMessages, defenseMsg)
		log.Debugf("[A2AS] Added in-context defense as system message")
	}

	// 注入 Codified Policies 作为系统消息
	if config.CodifiedPolicies.Enabled && config.CodifiedPolicies.Position == "as_system" && len(config.CodifiedPolicies.Policies) > 0 {
		policyMsg := map[string]interface{}{
			"role":    "system",
			"content": BuildPolicyBlock(config.CodifiedPolicies.Policies),
		}
		newMessages = append(newMessages, policyMsg)
		log.Debugf("[A2AS] Added %d codified policies as system message", len(config.CodifiedPolicies.Policies))
	}

	// 保留原始消息
	for _, msg := range rawMessages.Array() {
		message := parseMessage(msg)
		if message == nil {
			continue
		}
		newMessages = append(newMessages, message)
	}

	// 在用户消息前注入 In-Context Defenses
	if config.InContextDefenses.Enabled && config.InContextDefenses.Position == "before_user" {
		defenseContent := BuildDefenseBlock(config.InContextDefenses.Template)
		if config.InContextDefenses.Template == "custom" && config.InContextDefenses.CustomPrompt != "" {
			defenseContent = config.InContextDefenses.CustomPrompt
		}
		newMessages = insertBeforeUserMessages(newMessages, defenseContent)
		log.Debugf("[A2AS] Inserted in-context defense before user messages")
	}

	// 在用户消息前注入 Codified Policies
	if config.CodifiedPolicies.Enabled && config.CodifiedPolicies.Position == "before_user" && len(config.CodifiedPolicies.Policies) > 0 {
		newMessages = insertBeforeUserMessages(newMessages, BuildPolicyBlock(config.CodifiedPolicies.Policies))
		log.Debugf("[A2AS] Inserted codified policies before user messages")
	}

	messagesJSON, err := json.Marshal(newMessages)
	if err != nil {
		return body, err
	}

	newBody, err := sjson.SetRaw(string(body), "messages", string(messagesJSON))
	if err != nil {
		return body, err
	}

	return []byte(newBody), nil
}

func parseMessage(msg gjson.Result) map[string]interface{} {
	message := make(map[string]interface{})

	role := msg.Get("role").String()
	if role == "" {
		return nil
	}
	message["role"] = role

	content := msg.Get("content")
	if content.Exists() {
		if content.IsArray() {
			var contentArray []interface{}
			if err := json.Unmarshal([]byte(content.Raw), &contentArray); err == nil {
				message["content"] = contentArray
			}
		} else {
			message["content"] = content.String()
		}
	}

	// 保留其他字段（如 name, function_call, tool_calls 等）
	msg.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		if k != "role" && k != "content" {
			var v interface{}
			if err := json.Unmarshal([]byte(value.Raw), &v); err == nil {
				message[k] = v
			}
		}
		return true
	})

	return message
}

func insertBeforeUserMessages(messages []map[string]interface{}, contentToInsert string) []map[string]interface{} {
	if contentToInsert == "" {
		return messages
	}

	firstUserIndex := -1
	for i, msg := range messages {
		if role, ok := msg["role"].(string); ok && role == "user" {
			firstUserIndex = i
			break
		}
	}

	if firstUserIndex == -1 {
		return messages
	}

	newMessage := map[string]interface{}{
		"role":    "system",
		"content": contentToInsert,
	}

	result := make([]map[string]interface{}, 0, len(messages)+1)
	result = append(result, messages[:firstUserIndex]...)
	result = append(result, newMessage)
	result = append(result, messages[firstUserIndex:]...)

	return result
}

// verifyAndRemoveEmbeddedHashes 验证并移除 Prompt 中嵌入的 Hash 标记
// 格式：<a2as:TYPE:HASH>content</a2as:TYPE:HASH>
func verifyAndRemoveEmbeddedHashes(config AuthenticatedPromptsConfig, body []byte) ([]byte, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body, nil
	}

	var modifiedMessages []interface{}
	hasSignedMessage := false

	for _, msg := range messages.Array() {
		role := msg.Get("role").String()
		content := msg.Get("content").String()

		if content == "" {
			// 保留非文本消息
			var m interface{}
			if err := json.Unmarshal([]byte(msg.Raw), &m); err == nil {
				modifiedMessages = append(modifiedMessages, m)
			}
			continue
		}

		// 检查是否有嵌入的 Hash 标记
		verified, newContent, err := verifyEmbeddedHash(config, content)
		if err != nil {
			return nil, fmt.Errorf("message verification failed (role=%s): %w", role, err)
		}

		if verified {
			hasSignedMessage = true
		}

		// 构建修改后的消息
		message := make(map[string]interface{})
		message["role"] = role
		message["content"] = newContent

		// 保留其他字段
		msg.ForEach(func(key, value gjson.Result) bool {
			k := key.String()
			if k != "role" && k != "content" {
				var v interface{}
				if err := json.Unmarshal([]byte(value.Raw), &v); err == nil {
					message[k] = v
				}
			}
			return true
		})

		modifiedMessages = append(modifiedMessages, message)
	}

	// 如果启用了验签但没有找到任何签名，返回错误
	if !hasSignedMessage {
		return nil, fmt.Errorf("no signed messages found, but signature verification is enabled")
	}

	// 重建 JSON
	modifiedBody, err := sjson.SetBytes(body, "messages", modifiedMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild request body: %w", err)
	}

	return modifiedBody, nil
}

// verifyEmbeddedHash 验证单个内容中的嵌入 Hash
// 返回：(是否包含签名, 移除Hash后的内容, 错误)
func verifyEmbeddedHash(config AuthenticatedPromptsConfig, content string) (bool, string, error) {
	// 正则表达式匹配：<a2as:TYPE:HASH>content</a2as:TYPE:HASH>
	// TYPE 可以是 user, tool, system 等
	// HASH 是十六进制字符串
	// 注意：Go 不支持反向引用，所以需要手动验证闭合标签
	pattern := regexp.MustCompile(`<a2as:(\w+):([0-9a-fA-F]+)>(.*?)</a2as:(\w+):([0-9a-fA-F]+)>`)
	matches := pattern.FindStringSubmatch(content)

	if len(matches) == 0 {
		// 没有嵌入的 Hash，返回原内容
		return false, content, nil
	}

	if len(matches) != 6 {
		return false, "", fmt.Errorf("invalid a2as tag format")
	}

	openTagType := matches[1]
	openHash := matches[2]
	innerContent := matches[3]
	closeTagType := matches[4]
	closeHash := matches[5]

	// 验证开始和结束标签匹配
	if openTagType != closeTagType {
		return false, "", fmt.Errorf("tag type mismatch: open=%s, close=%s", openTagType, closeTagType)
	}
	if openHash != closeHash {
		return false, "", fmt.Errorf("hash mismatch in tags: open=%s, close=%s", openHash, closeHash)
	}

	// 计算期望的 Hash
	expectedHash := computeContentHash(config, innerContent)

	// 对比 Hash（不区分大小写）
	if !strings.EqualFold(openHash, expectedHash) {
		return false, "", fmt.Errorf("hash mismatch for type=%s (expected=%s, got=%s)",
			openTagType, expectedHash, openHash)
	}

	// 验证通过，返回移除 Hash 后的内容
	// 替换整个标记为内部内容
	newContent := pattern.ReplaceAllString(content, "$3")

	log.Debugf("[A2AS] Hash verified for type=%s, hash=%s", openTagType, openHash)

	return true, newContent, nil
}

// computeContentHash 计算内容的 HMAC-SHA256 Hash（截取配置的长度）
func computeContentHash(config AuthenticatedPromptsConfig, content string) string {
	// 解析 secret（支持 base64 或原始字符串）
	secretBytes, err := base64.StdEncoding.DecodeString(config.SharedSecret)
	if err != nil {
		secretBytes = []byte(config.SharedSecret)
	}

	// 计算 HMAC-SHA256
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(content))
	fullHash := hex.EncodeToString(mac.Sum(nil))

	// 截取指定长度
	if len(fullHash) > config.HashLength {
		return fullHash[:config.HashLength]
	}

	return fullHash
}

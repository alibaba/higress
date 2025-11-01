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
	"encoding/json"

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

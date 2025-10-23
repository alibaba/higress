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

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Metrics counters
const (
	metricA2ASRequestsTotal               = "a2as_requests_total"
	metricA2ASSignatureVerificationFailed = "a2as_signature_verification_failed"
	metricA2ASToolCallDenied              = "a2as_tool_call_denied"
	metricA2ASBoundariesApplied           = "a2as_security_boundaries_applied"
	metricA2ASDefensesInjected            = "a2as_defenses_injected"
	metricA2ASPoliciesInjected            = "a2as_policies_injected"
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

	if config.AuthenticatedPrompts.Enabled {
		if err := verifySignature(config.AuthenticatedPrompts, config.MaxRequestBodySize); err != nil {
			log.Errorf("[A2AS] Signature verification failed: %v", err)
			config.incrementMetric(metricA2ASSignatureVerificationFailed, 1)
			_ = proxywasm.SendHttpResponse(403, [][2]string{
				{"content-type", "application/json"},
			}, []byte(`{"error":"unauthorized","message":"Invalid or missing request signature"}`), -1)
			return types.ActionPause
		}
		log.Debugf("[A2AS] Signature verification passed")
	}

	proxywasm.RemoveHttpRequestHeader("content-length")

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, globalConfig A2ASConfig, body []byte) types.Action {
	globalConfig.incrementMetric(metricA2ASRequestsTotal, 1)

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
			[]byte(`{"error":"A2AS transformation failed"}`), -1)
		return types.ActionPause
	}

	if config.BehaviorCertificates.Enabled {
		if denied, tool := checkToolPermissions(config.BehaviorCertificates, modifiedBody); denied {
			log.Warnf("[A2AS] Tool call denied by behavior certificate: %s", tool)
			config.incrementMetric(metricA2ASToolCallDenied, 1)
			_ = proxywasm.SendHttpResponse(403, [][2]string{
				{"content-type", "application/json"},
			}, []byte(`{"error":"forbidden","message":"`+config.BehaviorCertificates.DenyMessage+`","denied_tool":"`+tool+`"}`), -1)
			return types.ActionPause
		}
	}

	if err := proxywasm.ReplaceHttpRequestBody(modifiedBody); err != nil {
		log.Errorf("[A2AS] Failed to replace request body: %v", err)
		return types.ActionContinue
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

	if config.InContextDefenses.Enabled && config.InContextDefenses.Position == "as_system" {
		defenseMsg := map[string]interface{}{
			"role":    "system",
			"content": BuildDefenseBlock(config.InContextDefenses.Template),
		}
		newMessages = append(newMessages, defenseMsg)
		config.incrementMetric(metricA2ASDefensesInjected, 1)
		log.Debugf("[A2AS] Added in-context defense as system message")
	}

	if config.CodifiedPolicies.Enabled && config.CodifiedPolicies.Position == "as_system" && len(config.CodifiedPolicies.Policies) > 0 {
		policyMsg := map[string]interface{}{
			"role":    "system",
			"content": BuildPolicyBlock(config.CodifiedPolicies.Policies),
		}
		newMessages = append(newMessages, policyMsg)
		config.incrementMetric(metricA2ASPoliciesInjected, 1)
		log.Debugf("[A2AS] Added %d codified policies as system message", len(config.CodifiedPolicies.Policies))
	}

	boundariesApplied := false
	for _, msg := range rawMessages.Array() {
		message := parseMessage(msg)
		if message == nil {
			continue
		}

		if config.SecurityBoundaries.Enabled {
			message = applySecurityBoundaries(config.SecurityBoundaries, message)
			boundariesApplied = true
		}

		newMessages = append(newMessages, message)
	}

	if boundariesApplied {
		config.incrementMetric(metricA2ASBoundariesApplied, 1)
	}

	if config.InContextDefenses.Enabled && config.InContextDefenses.Position == "before_user" {
		newMessages = insertBeforeUserMessages(newMessages, BuildDefenseBlock(config.InContextDefenses.Template))
		log.Debugf("[A2AS] Inserted in-context defense before user messages")
	}

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

func applySecurityBoundaries(config SecurityBoundariesConfig, message map[string]interface{}) map[string]interface{} {
	role, ok := message["role"].(string)
	if !ok {
		return message
	}

	content, ok := message["content"].(string)
	if !ok || content == "" {
		return message
	}

	var tagType string
	shouldWrap := false

	switch role {
	case "user":
		if config.WrapUserMessages {
			tagType = "user"
			shouldWrap = true
		}
	case "system":
		if config.WrapSystemMessages {
			tagType = "system"
			shouldWrap = true
		}
	case "tool", "function":
		if config.WrapToolOutputs {
			tagType = "tool"
			shouldWrap = true
		}
	}

	if shouldWrap {
		wrappedContent := WrapWithSecurityTag(content, tagType, config.IncludeContentDigest)
		message["content"] = wrappedContent
		log.Debugf("[A2AS] Wrapped %s message with security tag", role)
	}

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

func checkToolPermissions(config BehaviorCertificatesConfig, body []byte) (denied bool, toolName string) {
	toolChoice := gjson.GetBytes(body, "tool_choice")
	tools := gjson.GetBytes(body, "tools")

	if toolChoice.Exists() {
		if toolChoice.IsObject() {
			name := toolChoice.Get("function.name").String()
			if name != "" && !isToolAllowed(config.Permissions, name) {
				return true, name
			}
		}
	}

	if tools.Exists() && tools.IsArray() {
		for _, tool := range tools.Array() {
			name := tool.Get("function.name").String()
			if name != "" && !isToolAllowed(config.Permissions, name) {
				return true, name
			}
		}
	}

	return false, ""
}

func isToolAllowed(permissions AgentPermissions, toolName string) bool {
	for _, denied := range permissions.DeniedTools {
		if denied == toolName || denied == "*" {
			return false
		}
	}

	if len(permissions.AllowedTools) == 0 {
		return true
	}

	for _, allowed := range permissions.AllowedTools {
		if allowed == toolName || allowed == "*" {
			return true
		}
	}

	return false
}

func verifySignature(config AuthenticatedPromptsConfig, maxBodySize int) error {
	switch config.Mode {
	case "rfc9421":
		log.Debugf("[A2AS] Using RFC 9421 signature verification mode")
		return verifyRFC9421Signature(config)
	
	case "simple":
		log.Debugf("[A2AS] Using simple HMAC signature verification mode")
		return verifySimpleSignature(config, maxBodySize)
	
	default:
		return fmt.Errorf("unsupported signature mode: %s", config.Mode)
	}
}

func verifySimpleSignature(config AuthenticatedPromptsConfig, maxBodySize int) error {
	signatureHeader, err := proxywasm.GetHttpRequestHeader(config.SignatureHeader)
	
	if err != nil || signatureHeader == "" {
		if config.AllowUnsigned {
			log.Debugf("[A2AS] No signature found, but allowUnsigned=true, continuing")
			return nil
		}
		return fmt.Errorf("missing signature header '%s'", config.SignatureHeader)
	}

	if config.SharedSecret == "" {
		log.Warnf("[A2AS] Signature header present but no sharedSecret configured, skipping verification")
		return nil
	}

	body, err := proxywasm.GetHttpRequestBody(0, maxBodySize)
	if err != nil {
		return fmt.Errorf("failed to get request body for signature verification: %v", err)
	}

	secretBytes, err := base64.StdEncoding.DecodeString(config.SharedSecret)
	if err != nil {
		secretBytes = []byte(config.SharedSecret)
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	expectedSignatureBase64 := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if signatureHeader != expectedSignature && signatureHeader != expectedSignatureBase64 {
		log.Errorf("[A2AS] Signature verification failed")
		return fmt.Errorf("invalid signature")
	}

	log.Debugf("[A2AS] Signature verification passed")
	return nil
}

package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

type MessageInfo struct {
	Index        int
	Role         string
	Content      string
	ImageFingerprint string // hash fingerprint of image_url contents for cache key differentiation
}

func GenerateHexID(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateRandomChatID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 29)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return "chatcmpl-" + string(b)
}

func ExtractMessageFromStreamingBody(data []byte, jsonPath string) string {
	chunks := bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
	strChunks := []string{}
	for _, chunk := range chunks {
		// Example: "choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]
		strChunks = append(strChunks, gjson.GetBytes(chunk, jsonPath).String())
	}
	return strings.Join(strChunks, "")
}

func GetConsumer(ctx wrapper.HttpContext) string {
	return ctx.GetStringContext("consumer", "")
}

// ParseAllMessages extracts all messages from an OpenAI-format request body.
// Handles both string and array content formats.
// For multi-modal messages, also computes an image fingerprint for cache key differentiation.
func ParseAllMessages(body []byte) []MessageInfo {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return nil
	}
	var result []MessageInfo
	for i, msg := range messages.Array() {
		role := msg.Get("role").String()
		content, imageFingerprint := extractContentAndImageFingerprint(msg.Get("content"))
		result = append(result, MessageInfo{
			Index:            i,
			Role:             role,
			Content:          content,
			ImageFingerprint: imageFingerprint,
		})
	}
	return result
}

// extractContentAndImageFingerprint extracts text content and computes a fingerprint
// of all image_url entries. The fingerprint is a hex-encoded SHA256 hash of all image
// URLs/base64 data concatenated, or empty if no images are present.
func extractContentAndImageFingerprint(content gjson.Result) (string, string) {
	if !content.IsArray() {
		return content.String(), ""
	}
	var textParts []string
	var imageParts []string
	for _, item := range content.Array() {
		switch item.Get("type").String() {
		case "text":
			textParts = append(textParts, item.Get("text").String())
		case "image_url":
			imgURL := item.Get("image_url.url").String()
			if imgURL != "" {
				imageParts = append(imageParts, imgURL)
			}
		}
	}
	text := strings.Join(textParts, "")
	imageFingerprint := ""
	if len(imageParts) > 0 {
		hash := sha256.Sum256([]byte(strings.Join(imageParts, "\n")))
		imageFingerprint = hex.EncodeToString(hash[:16])
	}
	return text, imageFingerprint
}

// FilterByRole returns messages that match ANY of the given roles.
// If no roles are specified, all messages are returned.
func FilterByRole(messages []MessageInfo, roles ...string) []MessageInfo {
	if len(roles) == 0 {
		return messages
	}
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	var result []MessageInfo
	for _, msg := range messages {
		if roleSet[msg.Role] {
			result = append(result, msg)
		}
	}
	return result
}

// BuildRedisKeys generates Redis keys for each message using SHA256 hash.
// Uses {ai_sec} hash-tag prefix for Redis Cluster slot affinity.
// Includes consumer and policyFingerprint in the hash to prevent cross-consumer/cross-policy cache pollution.
// policyFingerprint should encode policy dimensions that affect check results (e.g. action, checkService, riskLevelBar).
// Includes image fingerprint in the hash to differentiate multi-modal messages with different images.
func BuildRedisKeys(messages []MessageInfo, consumer string, policyFingerprint string) []string {
	keys := make([]string, len(messages))
	for i, msg := range messages {
		raw := fmt.Sprintf("%s:%s:%s:%s:%s", policyFingerprint, consumer, msg.Role, msg.Content, msg.ImageFingerprint)
		hash := sha256.Sum256([]byte(raw))
		keys[i] = fmt.Sprintf("{ai_sec}:%s", hex.EncodeToString(hash[:16]))
	}
	return keys
}

// FilterUnchecked returns messages whose corresponding Redis MGet result is null.
// If the MGet response is an error or not a valid array, all messages are treated as unchecked.
func FilterUnchecked(messages []MessageInfo, redisResponse resp.Value) []MessageInfo {
	if redisResponse.Error() != nil {
		log.Warnf("MGet returned error response: %v, treating all messages as unchecked", redisResponse.Error())
		return messages
	}
	arr := redisResponse.Array()
	var unchecked []MessageInfo
	for i, msg := range messages {
		if i >= len(arr) || arr[i].IsNull() {
			unchecked = append(unchecked, msg)
		}
	}
	return unchecked
}

// ConcatTextContent joins message text contents with newline, skipping empty content.
func ConcatTextContent(messages []MessageInfo) string {
	var parts []string
	for _, msg := range messages {
		if msg.Content != "" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n")
}

// MarkChecked sets Redis keys for all messages with the given TTL.
// Tries EVAL (Lua script) first for atomicity; falls back to sequential SetEx on failure.
func MarkChecked(client wrapper.RedisClient, messages []MessageInfo, consumer string, policyFingerprint string, ttl int, callback func()) {
	if len(messages) == 0 {
		callback()
		return
	}
	keys := BuildRedisKeys(messages, consumer, policyFingerprint)
	ikeys := make([]interface{}, len(keys))
	for i, k := range keys {
		ikeys[i] = k
	}
	args := []interface{}{ttl}
	script := `for i=1,#KEYS do redis.call('SETEX',KEYS[i],ARGV[1],'1') end return 1`
	err := client.Eval(script, len(keys), ikeys, args, func(response resp.Value) {
		if response.Error() != nil {
			log.Warnf("EVAL failed at Redis side: %v, falling back to sequential SetEx", response.Error())
			markCheckedSequential(client, keys, 0, ttl, callback)
			return
		}
		log.Infof("MarkChecked via EVAL succeeded for %d keys", len(keys))
		callback()
	})
	if err != nil {
		log.Warnf("failed to dispatch EVAL: %v, falling back to sequential SetEx", err)
		markCheckedSequential(client, keys, 0, ttl, callback)
	}
}

func markCheckedSequential(client wrapper.RedisClient, keys []string, idx int, ttl int, callback func()) {
	if idx >= len(keys) {
		callback()
		return
	}
	err := client.SetEx(keys[idx], "1", ttl, func(response resp.Value) {
		if response.Error() != nil {
			log.Warnf("SetEx failed for key %s: %v", keys[idx], response.Error())
		}
		markCheckedSequential(client, keys, idx+1, ttl, callback)
	})
	if err != nil {
		log.Warnf("failed to dispatch SetEx for key %s: %v", keys[idx], err)
		markCheckedSequential(client, keys, idx+1, ttl, callback)
	}
}

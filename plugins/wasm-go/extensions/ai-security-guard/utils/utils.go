package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	mrand "math/rand"
	"strings"

	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

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
	return ctx.GetStringContext("x-mse-consumer", "")
}

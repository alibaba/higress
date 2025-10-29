package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	mrand "math/rand"
	"net/url"
	"sort"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

func urlEncoding(rawStr string) string {
	encodedStr := url.PathEscape(rawStr)
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%2B")
	encodedStr = strings.ReplaceAll(encodedStr, ":", "%3A")
	encodedStr = strings.ReplaceAll(encodedStr, "=", "%3D")
	encodedStr = strings.ReplaceAll(encodedStr, "&", "%26")
	encodedStr = strings.ReplaceAll(encodedStr, "$", "%24")
	encodedStr = strings.ReplaceAll(encodedStr, "@", "%40")
	return encodedStr
}

func hmacSha1(message, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	hash := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

func GetSign(params map[string]string, secret string) string {
	paramArray := []string{}
	for k, v := range params {
		paramArray = append(paramArray, urlEncoding(k)+"="+urlEncoding(v))
	}
	sort.Slice(paramArray, func(i, j int) bool {
		return paramArray[i] <= paramArray[j]
	})
	canonicalStr := strings.Join(paramArray, "&")
	signStr := "POST&%2F&" + urlEncoding(canonicalStr)
	proxywasm.LogDebugf("String to sign is: %s", signStr)
	return hmacSha1(signStr, secret)
}

func GenerateHexID(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 29)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return "chatcmpl-" + string(b)
}

package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func Sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func Hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}

/**
 * @param secretId 秘钥id
 * @param secretKey 秘钥
 * @param timestamp 时间戳
 * @param host 目标域名
 * @param action 请求动作
 * @param payload 请求体
 * @return 签名
 */
func GetTC3Authorizationcode(secretId string, secretKey string, timestamp int64, host string, action string, payload string) string {
	algorithm := "TC3-HMAC-SHA256"
	service := "hunyuan" // 注意，必须和域名中的产品名保持一致

	// step 1: build canonical request string
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"

	// fmt.Println("payload is: %s", payload)
	hashedRequestPayload := Sha256hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)
	// fmt.Println(canonicalRequest)

	// step 2: build string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := Sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)
	// fmt.Println(string2sign)

	// step 3: sign string
	secretDate := Hmacsha256(date, "TC3"+secretKey)
	secretService := Hmacsha256(service, secretDate)
	secretSigning := Hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(Hmacsha256(string2sign, secretSigning)))
	// fmt.Println(signature)

	// step 4: build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		secretId,
		credentialScope,
		signedHeaders,
		signature)

	curl := fmt.Sprintf(`curl -X POST https://%s \
		-H "Authorization: %s" \
		-H "Content-Type: application/json" \
		-H "Host: %s" -H "X-TC-Action: %s" \
		-H "X-TC-Timestamp: %d" \
		-H "X-TC-Version: 2023-09-01" \
		-d '%s'`, host, authorization, host, action, timestamp, payload)
	fmt.Println(curl)

	return authorization
}

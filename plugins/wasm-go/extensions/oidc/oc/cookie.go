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

package oc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

type CookieData struct {
	IDToken   string
	Secret    string
	Nonce     []byte
	CreatedAt time.Time
	ExpiresOn time.Time
}

type CookieOption struct {
	Name     string
	Domain   string
	Secret   string
	value    string
	Path     string
	SameSite string
	Expire   time.Time
	Secure   bool
	HTTPOnly bool
}

// SerializeAndEncrypt 将 CookieData 对象序列化并加密为一个安全的cookie header
func SerializeAndEncryptCookieData(data *CookieData, keySecret string, cookieSettings *CookieOption) (string, error) {
	return buildSecureCookieHeader(data, keySecret, cookieSettings)
}

// DeserializeCookieData 将一个安全的cookie header解密并反序列化为 CookieData 对象
func DeserializeCookieData(cookievalue string) (*CookieData, error) {

	data, err := retrieveCookieData(cookievalue)
	if err != nil {
		return nil, err
	}
	if checkCookieExpiry(data) {
		return nil, fmt.Errorf("cookie is expired")
	}
	return data, nil
}
func Set32Bytes(key string) string {
	const desiredLength = 32
	keyLength := len(key)

	var adjustedKey string
	if keyLength > desiredLength {
		adjustedKey = key[:desiredLength]
	} else if keyLength < desiredLength {
		padding := strings.Repeat("0", desiredLength-keyLength)
		adjustedKey = key + padding
	} else {
		adjustedKey = key
	}
	return adjustedKey
}

// 必须是16/24/32字节长
func Decrypt(ciphertext string, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	decodedCiphertext, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	if len(decodedCiphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext is too short")
	}

	iv := decodedCiphertext[:aes.BlockSize]
	decodedCiphertext = decodedCiphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(decodedCiphertext, decodedCiphertext)

	return string(decodedCiphertext), nil
}

func encrypt(plainText string, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plainText))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plainText))

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func buildSecureCookieHeader(data *CookieData, keySecret string, cookieSettings *CookieOption) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	encryptedValue, err := encrypt(string(jsonData), keySecret)
	if err != nil {
		return "", err
	}

	encodedValue := url.QueryEscape(encryptedValue)
	cookieSettings.value = encodedValue

	return generateCookie(cookieSettings), nil
}

func retrieveCookieData(cookieValue string) (*CookieData, error) {
	var data CookieData
	err := json.Unmarshal([]byte(cookieValue), &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func generateCookie(settings *CookieOption) string {
	var secureFlag, httpOnlyFlag, sameSiteFlag string
	if settings.Secure {
		secureFlag = "Secure;"
	}

	if settings.HTTPOnly {
		httpOnlyFlag = "HttpOnly;"
	}

	if settings.SameSite != "" {
		sameSiteFlag = fmt.Sprintf("SameSite=%s;", settings.SameSite)
	}

	expiresStr := settings.Expire.Format(time.RFC1123)
	maxAge := int(settings.Expire.Sub(time.Now()).Seconds())

	cookie := fmt.Sprintf("%s=%s; Path=%s; Domain=%s; Expires=%s; Max-Age=%d; %s %s %s",
		settings.Name,
		settings.value,
		settings.Path,
		settings.Domain,
		expiresStr,
		maxAge,
		secureFlag,
		httpOnlyFlag,
		sameSiteFlag,
	)
	return cookie
}

func checkCookieExpiry(data *CookieData) bool {
	return data.ExpiresOn.Before(time.Now())
}

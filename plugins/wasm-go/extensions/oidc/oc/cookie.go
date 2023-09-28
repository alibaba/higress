package oc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"time"
)

var encryptionKey = []byte("avery strongkey!")

// 必须是16/24/32字节长
func Decrypt(ciphertext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
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

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(decodedCiphertext, decodedCiphertext)

	return string(decodedCiphertext), nil
}

func encrypt(plainText string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
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

func buildSecureCookieHeader(value, domain string, expires time.Time, onSecure bool) (string, error) {
	// 加密cookie值
	encryptedValue, err := encrypt(value)
	if err != nil {
		return "", err
	}

	encodedValue := url.QueryEscape(encryptedValue)

	cookie := generateCookie("oidc_oauth2_wasm_plugin", encodedValue, domain, expires, onSecure)
	return cookie, nil
}

func generateCookie(name, value, domain string, expires time.Time, onSecure bool) string {
	var secureFlag, httpOnlyFlag, sameSiteFlag string
	if onSecure {
		secureFlag = "Secure;"
	}

	expiresStr := expires.Format(time.RFC1123)

	maxAge := int(expires.Sub(time.Now()).Seconds())

	httpOnlyFlag = "HttpOnly;"
	sameSiteFlag = "SameSite=Lax;"

	cookie := fmt.Sprintf("%s=%s; Path=/; Domain=%s; Expires=%s; Max-Age=%d; %s %s %s",
		name,
		value,
		domain,
		expiresStr,
		maxAge,
		secureFlag,
		httpOnlyFlag,
		sameSiteFlag,
	)
	return cookie
}

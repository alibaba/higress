package oc

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"strconv"
	"strings"
	"time"
)

const (
	// StateSigningKey is used to sign the state parameter. This should be kept secret.
	StateSigningKey = "your-secret-key"
	ExpiryDuration  = time.Hour
)

func GenState() string {
	nonce, _ := Nonce(16)
	expiry := time.Now().Add(ExpiryDuration).Unix()
	state := fmt.Sprintf("%s:%d:%d", base64.RawURLEncoding.EncodeToString(nonce), time.Now().Unix(), expiry)
	signature := SignState(state)
	return fmt.Sprintf("%s.%s", state, signature)
}

// SignState signs the state using HMAC.
func SignState(state string) string {
	mac := hmac.New(sha256.New, []byte(StateSigningKey))
	mac.Write([]byte(state))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature)
}

func VerifyState(state, signature string) bool {
	expectedSignature := SignState(state)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return false
	}
	parts := strings.Split(state, ":")
	if len(parts) != 3 {
		return false
	}

	expiry, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix() > expiry {
		return false
	}
	return true
}

func Nonce(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func SetNonce(nonce string) oauth2.AuthCodeOption {
	return oauth2.SetAuthURLParam("nonce", nonce)
}

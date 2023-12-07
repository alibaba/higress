// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oc

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"strings"
)

func Nonce(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	return b, err
}

func HashNonce(nonce []byte) string {
	hasher := sha256.New()
	hasher.Write(nonce)
	return base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
}

func GenState(nonce []byte, key string, redirectUrl string) string {
	hashedNonce := HashNonce(nonce)
	encodedRedirectUrl := base64.RawURLEncoding.EncodeToString([]byte(redirectUrl))
	state := fmt.Sprintf("%s:%s", hashedNonce, encodedRedirectUrl)
	signature := SignState(state, key)
	return fmt.Sprintf("%s.%s", state, signature)
}

func SignState(state string, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(state))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyState(state, signature, key, redirect string) error {
	if !hmac.Equal([]byte(signature), []byte(SignState(state, key))) {
		return fmt.Errorf("signature mismatch")
	}

	parts := strings.Split(state, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid state format")
	}

	redirectUrl, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode redirect URL: %v", err)
	}
	if string(redirectUrl) != redirect {
		return fmt.Errorf("redirect URL mismatch")
	}

	return nil
}

func SetNonce(nonce string) oauth2.AuthCodeOption {
	return oauth2.SetAuthURLParam("nonce", nonce)
}

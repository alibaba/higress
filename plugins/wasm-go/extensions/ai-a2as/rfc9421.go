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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

type RFC9421SignatureParams struct {
	SignatureID    string
	Components     []string
	Created        int64
	Expires        int64
	KeyID          string
	Algorithm      string
	Nonce          string
	SignatureValue string
}

func verifyRFC9421Signature(config AuthenticatedPromptsConfig) error {
	// 自动添加Content-Digest头（如果客户端没有提供）
	if err := ensureContentDigest(); err != nil {
		return fmt.Errorf("failed to ensure Content-Digest: %v", err)
	}

	signatureInputHeader, err := proxywasm.GetHttpRequestHeader("Signature-Input")
	if err != nil || signatureInputHeader == "" {
		return fmt.Errorf("missing Signature-Input header")
	}

	params, err := parseSignatureInput(signatureInputHeader)
	if err != nil {
		return fmt.Errorf("failed to parse Signature-Input: %v", err)
	}

	signatureHeader, err := proxywasm.GetHttpRequestHeader(config.SignatureHeader)
	if err != nil || signatureHeader == "" {
		return fmt.Errorf("missing Signature header")
	}

	sigValue, err := parseSignatureHeader(signatureHeader, params.SignatureID)
	if err != nil {
		return fmt.Errorf("failed to parse Signature header: %v", err)
	}
	params.SignatureValue = sigValue

	if err := validateSignatureParams(params, config); err != nil {
		return err
	}

	signatureBase, err := buildSignatureBase(params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %v", err)
	}

	if err := validateSignature(signatureBase, params.SignatureValue, config.SharedSecret); err != nil {
		return err
	}

	return nil
}

func parseSignatureInput(input string) (*RFC9421SignatureParams, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Signature-Input format")
	}

	params := &RFC9421SignatureParams{
		SignatureID: strings.TrimSpace(parts[0]),
	}

	content := parts[1]

	componentsMatch := regexp.MustCompile(`\((.*?)\)`).FindStringSubmatch(content)
	if len(componentsMatch) < 2 {
		return nil, fmt.Errorf("missing components in Signature-Input")
	}

	componentsStr := componentsMatch[1]
	components := strings.Split(componentsStr, " ")
	for _, comp := range components {
		comp = strings.Trim(comp, `"`)
		if comp != "" {
			params.Components = append(params.Components, comp)
		}
	}

	paramPattern := regexp.MustCompile(`(\w+)=([^;]+)`)
	matches := paramPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := match[1]
		value := strings.Trim(match[2], `"`)

		switch key {
		case "created":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				params.Created = v
			}
		case "expires":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				params.Expires = v
			}
		case "keyid":
			params.KeyID = value
		case "alg":
			params.Algorithm = value
		case "nonce":
			params.Nonce = value
		}
	}

	return params, nil
}

func parseSignatureHeader(header, signatureID string) (string, error) {
	prefix := signatureID + "=:"
	if !strings.Contains(header, prefix) {
		return "", fmt.Errorf("signature ID '%s' not found in Signature header", signatureID)
	}

	startIdx := strings.Index(header, prefix) + len(prefix)
	endIdx := strings.Index(header[startIdx:], ":")
	if endIdx == -1 {
		return "", fmt.Errorf("invalid Signature format")
	}

	sigValue := header[startIdx : startIdx+endIdx]
	return sigValue, nil
}

func validateSignatureParams(params *RFC9421SignatureParams, config AuthenticatedPromptsConfig) error {
	now := time.Now().Unix()

	if params.Created == 0 {
		return fmt.Errorf("missing 'created' parameter in Signature-Input")
	}

	clockSkew := int64(config.ClockSkew)
	if params.Created > now+clockSkew {
		return fmt.Errorf("signature created in the future (clock skew exceeded)")
	}

	age := now - params.Created
	maxAge := int64(config.RFC9421.MaxAge)
	if maxAge > 0 && age > maxAge {
		return fmt.Errorf("signature too old (age: %d seconds, max: %d seconds)", age, maxAge)
	}

	// 验证 expires（如果启用）
	if config.RFC9421.EnforceExpires && params.Expires > 0 {
		if now > params.Expires {
			return fmt.Errorf("signature expired")
		}
	}

	// 验证算法
	if params.Algorithm != "" && params.Algorithm != config.Algorithm {
		return fmt.Errorf("unsupported algorithm: %s", params.Algorithm)
	}

	// 验证必需组件
	if len(config.RFC9421.RequiredComponents) > 0 {
		for _, required := range config.RFC9421.RequiredComponents {
			found := false
			for _, comp := range params.Components {
				if comp == required {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing required component: %s", required)
			}
		}
	}

	return nil
}

func buildSignatureBase(params *RFC9421SignatureParams) (string, error) {
	var lines []string

	for _, component := range params.Components {
		var value string
		var err error

		if strings.HasPrefix(component, "@") {
			value, err = getDerivedComponent(component)
		} else {
			value, err = getHeaderComponent(component)
		}

		if err != nil {
			return "", fmt.Errorf("failed to get component '%s': %v", component, err)
		}

		lines = append(lines, fmt.Sprintf(`"%s": %s`, component, value))
	}

	sigParams := buildSignatureParamsLine(params)
	lines = append(lines, sigParams)

	return strings.Join(lines, "\n"), nil
}

func getDerivedComponent(component string) (string, error) {
	switch component {
	case "@method":
		method, err := proxywasm.GetHttpRequestHeader(":method")
		if err != nil {
			return "", err
		}
		return method, nil

	case "@path":
		path, err := proxywasm.GetHttpRequestHeader(":path")
		if err != nil {
			return "", err
		}
		return path, nil

	case "@authority":
		authority, err := proxywasm.GetHttpRequestHeader(":authority")
		if err != nil {
			return "", err
		}
		return authority, nil

	case "@request-target":
		method, _ := proxywasm.GetHttpRequestHeader(":method")
		path, _ := proxywasm.GetHttpRequestHeader(":path")
		return fmt.Sprintf("%s %s", method, path), nil

	default:
		return "", fmt.Errorf("unsupported derived component: %s", component)
	}
}

func getHeaderComponent(component string) (string, error) {
	if component == "content-digest" {
		return getContentDigestValue()
	}

	value, err := proxywasm.GetHttpRequestHeader(component)
	if err != nil {
		return "", err
	}
	return value, nil
}

// ensureContentDigest 自动添加Content-Digest头（如果客户端未提供）
func ensureContentDigest() error {
	existingDigest, err := proxywasm.GetHttpRequestHeader("Content-Digest")
	if err == nil && existingDigest != "" {
		return nil
	}

	body, err := proxywasm.GetHttpRequestBody(0, 10*1024*1024)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}

	hash := sha256.Sum256(body)
	digestValue := base64.StdEncoding.EncodeToString(hash[:])
	contentDigest := fmt.Sprintf("sha-256=:%s:", digestValue)

	if err := proxywasm.AddHttpRequestHeader("Content-Digest", contentDigest); err != nil {
		return fmt.Errorf("failed to add Content-Digest header: %v", err)
	}

	return nil
}

func getContentDigestValue() (string, error) {
	providedDigest, err := proxywasm.GetHttpRequestHeader("Content-Digest")

	body, err := proxywasm.GetHttpRequestBody(0, 10*1024*1024)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(body)
	digestValue := base64.StdEncoding.EncodeToString(hash[:])
	actualDigest := fmt.Sprintf("sha-256=:%s:", digestValue)

	if providedDigest != "" {
		if providedDigest != actualDigest {
			return "", fmt.Errorf("Content-Digest mismatch: provided=%s, actual=%s", providedDigest, actualDigest)
		}
		return providedDigest, nil
	}

	return actualDigest, nil
}

func buildSignatureParamsLine(params *RFC9421SignatureParams) string {
	components := make([]string, len(params.Components))
	for i, comp := range params.Components {
		components[i] = fmt.Sprintf(`"%s"`, comp)
	}
	componentsList := strings.Join(components, " ")

	var paramsParts []string
	paramsParts = append(paramsParts, fmt.Sprintf("(%s)", componentsList))

	if params.Created > 0 {
		paramsParts = append(paramsParts, fmt.Sprintf("created=%d", params.Created))
	}
	if params.Expires > 0 {
		paramsParts = append(paramsParts, fmt.Sprintf("expires=%d", params.Expires))
	}
	if params.KeyID != "" {
		paramsParts = append(paramsParts, fmt.Sprintf("keyid=\"%s\"", params.KeyID))
	}
	if params.Algorithm != "" {
		paramsParts = append(paramsParts, fmt.Sprintf("alg=\"%s\"", params.Algorithm))
	}
	if params.Nonce != "" {
		paramsParts = append(paramsParts, fmt.Sprintf("nonce=\"%s\"", params.Nonce))
	}

	paramsStr := strings.Join(paramsParts, ";")
	return fmt.Sprintf(`"@signature-params": %s`, paramsStr)
}

func validateSignature(signatureBase, signatureValue, sharedSecret string) error {
	secretBytes, err := base64.StdEncoding.DecodeString(sharedSecret)
	if err != nil {
		secretBytes = []byte(sharedSecret)
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signatureBase))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if signatureValue != expectedSignature {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

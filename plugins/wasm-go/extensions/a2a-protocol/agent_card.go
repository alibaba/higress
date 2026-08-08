// Copyright (c) 2026 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	canonicalAgentCardPath = "/.well-known/agent-card.json"
	legacyAgentCardPath    = "/.well-known/agent.json"
	defaultMaxCardBytes    = 256 << 10
	hardMaxCardBytes       = 1 << 20

	cardSignaturePreserve = "preserve"
	cardSignatureResign   = "resign"
)

var (
	errInvalidAgentCard = errors.New("invalid A2A Agent Card")
	errUnsafeEndpoint   = errors.New("unsafe A2A Agent Card endpoint")
)

type cardResult struct {
	body      []byte
	rewritten bool
}

func processAgentCard(body []byte, config pluginConfig, legacy bool, externalURL string) (cardResult, error) {
	if len(body) > config.AgentCard.MaxResponseBytes {
		return cardResult{}, fmt.Errorf("%w: response exceeds configured limit", errInvalidAgentCard)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var card map[string]json.RawMessage
	if err := decoder.Decode(&card); err != nil || card == nil {
		return cardResult{}, fmt.Errorf("%w: response must be a JSON object", errInvalidAgentCard)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return cardResult{}, fmt.Errorf("%w: trailing JSON data", errInvalidAgentCard)
	}
	if legacy {
		if err := validateAndRewriteLegacyCard(card, externalURL, config.AgentCard.rewrite); err != nil {
			return cardResult{}, err
		}
	} else if err := validateAndRewriteV1Card(card, externalURL, config.AgentCard.rewrite); err != nil {
		return cardResult{}, err
	}
	if hasCardSignature(card) && config.AgentCard.rewrite {
		switch config.AgentCard.SignatureMode {
		case cardSignaturePreserve:
			return cardResult{body: body}, nil
		case cardSignatureResign:
			return cardResult{}, fmt.Errorf("%w: signature resigning is not configured", errInvalidAgentCard)
		}
	}
	if !config.AgentCard.rewrite {
		return cardResult{body: body}, nil
	}
	rewritten, err := json.Marshal(card)
	if err != nil {
		return cardResult{}, fmt.Errorf("%w: encode rewritten response", errInvalidAgentCard)
	}
	return cardResult{body: rewritten, rewritten: true}, nil
}

func validateAndRewriteV1Card(card map[string]json.RawMessage, externalURL string, rewrite bool) error {
	raw, ok := card["supportedInterfaces"]
	if !ok {
		return fmt.Errorf("%w: supportedInterfaces is required", errInvalidAgentCard)
	}
	var interfaces []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &interfaces); err != nil || len(interfaces) == 0 {
		return fmt.Errorf("%w: supportedInterfaces must be a non-empty array", errInvalidAgentCard)
	}
	for i, iface := range interfaces {
		endpoint, err := requiredString(iface, "url")
		if err != nil {
			return fmt.Errorf("%w: supportedInterfaces[%d].url", errInvalidAgentCard, i)
		}
		if err := validateHTTPSURL(endpoint); err != nil {
			return fmt.Errorf("supportedInterfaces[%d]: %w", i, err)
		}
		transport, err := requiredString(iface, "transport")
		if err != nil || !isJSONRPCTransport(transport) {
			return fmt.Errorf("%w: supportedInterfaces[%d] uses unsupported transport", errInvalidAgentCard, i)
		}
		if rewrite {
			iface["url"], _ = json.Marshal(externalURL)
		}
	}
	if rewrite {
		card["supportedInterfaces"], _ = json.Marshal(interfaces)
	}
	return nil
}

func validateAndRewriteLegacyCard(card map[string]json.RawMessage, externalURL string, rewrite bool) error {
	endpoint, err := requiredString(card, "url")
	if err != nil {
		return fmt.Errorf("%w: url is required", errInvalidAgentCard)
	}
	if err := validateHTTPSURL(endpoint); err != nil {
		return err
	}
	if raw, ok := card["preferredTransport"]; ok {
		var transport string
		if json.Unmarshal(raw, &transport) != nil || !isJSONRPCTransport(transport) {
			return fmt.Errorf("%w: unsupported preferredTransport", errInvalidAgentCard)
		}
	}
	if rewrite {
		card["url"], _ = json.Marshal(externalURL)
	}
	return nil
}

func requiredString(object map[string]json.RawMessage, key string) (string, error) {
	raw, ok := object[key]
	if !ok {
		return "", errInvalidAgentCard
	}
	var value string
	if json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" {
		return "", errInvalidAgentCard
	}
	return value, nil
}

func isJSONRPCTransport(transport string) bool {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(transport), "-", ""))
	return normalized == "JSONRPC"
}

func validateHTTPSURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" {
		return fmt.Errorf("%w: endpoint must be an absolute HTTPS URL", errUnsafeEndpoint)
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("%w: credentials and fragments are forbidden", errUnsafeEndpoint)
	}
	if err := validatePort(parsed.Port()); err != nil {
		return err
	}
	return validatePublicHost(parsed.Hostname())
}

func validatePublicHost(host string) error {
	normalized := strings.TrimSuffix(strings.ToLower(host), ".")
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return fmt.Errorf("%w: loopback host is forbidden", errUnsafeEndpoint)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("%w: loopback, link-local, and unspecified addresses are forbidden", errUnsafeEndpoint)
		}
	}
	return nil
}

func validatePort(port string) error {
	if port == "" {
		return nil
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("%w: invalid endpoint port", errUnsafeEndpoint)
	}
	return nil
}

func hasCardSignature(card map[string]json.RawMessage) bool {
	raw, ok := card["signatures"]
	if !ok {
		return false
	}
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) && !bytes.Equal(trimmed, []byte("[]"))
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing interface{}
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func deriveExternalAgentURL(ctx requestContext, config pluginConfig, requestPath string) (string, error) {
	if config.Agent.ExternalBaseURL != "" {
		if err := validateHTTPSURL(config.Agent.ExternalBaseURL); err != nil {
			return "", fmt.Errorf("configured agent.externalBaseURL: %w", err)
		}
		return strings.TrimRight(config.Agent.ExternalBaseURL, "/"), nil
	}
	scheme := firstForwardedValue(ctx.header("x-forwarded-proto"))
	if scheme == "" {
		scheme = ctx.scheme()
	}
	authority := firstForwardedValue(ctx.header("x-forwarded-host"))
	if authority == "" {
		authority = ctx.host()
	}
	candidate := strings.ToLower(scheme) + "://" + authority
	if err := validateHTTPSURL(candidate); err != nil {
		return "", fmt.Errorf("request-visible gateway endpoint: %w", err)
	}
	parsedCandidate, err := url.Parse(candidate)
	if err != nil || parsedCandidate.EscapedPath() != "" || parsedCandidate.RawQuery != "" || parsedCandidate.Fragment != "" {
		return "", fmt.Errorf("request-visible gateway authority is malformed")
	}
	routePrefix := stripAgentCardSuffix(requestPath, config.AgentCard.Path)
	if routePrefix == "/" {
		routePrefix = ""
	}
	return strings.TrimRight(candidate, "/") + routePrefix, nil
}

type requestContext interface {
	scheme() string
	host() string
	header(string) string
}

func firstForwardedValue(value string) string {
	if index := strings.IndexByte(value, ','); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func stripAgentCardSuffix(requestPath, configuredPath string) string {
	if index := strings.IndexByte(requestPath, '?'); index >= 0 {
		requestPath = requestPath[:index]
	}
	for _, suffix := range []string{configuredPath, canonicalAgentCardPath, legacyAgentCardPath} {
		if strings.HasSuffix(requestPath, suffix) {
			prefix := strings.TrimSuffix(requestPath, suffix)
			if prefix == "" {
				return "/"
			}
			return strings.TrimRight(prefix, "/")
		}
	}
	return "/"
}

func rewrittenETag(body []byte) string {
	digest := sha256.Sum256(body)
	return `"` + hex.EncodeToString(digest[:]) + `"`
}

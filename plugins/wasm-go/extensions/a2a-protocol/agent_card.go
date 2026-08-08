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
	signed := hasCardSignature(card)
	if signed && config.AgentCard.rewrite && config.AgentCard.SignatureMode == cardSignatureResign {
		return cardResult{}, fmt.Errorf("%w: signature resigning is not configured", errInvalidAgentCard)
	}
	exposeOriginalEndpoints := !config.AgentCard.rewrite || signed
	legacy, err := detectLegacyCard(card, config.Legacy03.Enabled, legacy)
	if err != nil {
		return cardResult{}, err
	}
	if legacy {
		if err := validateAndRewriteLegacyCard(card, externalURL, config.AgentCard.rewrite, exposeOriginalEndpoints); err != nil {
			return cardResult{}, err
		}
	} else if err := validateAndRewriteV1Card(card, externalURL, config.AgentCard.rewrite, exposeOriginalEndpoints); err != nil {
		return cardResult{}, err
	}
	if signed && config.AgentCard.rewrite {
		return cardResult{body: body}, nil
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

func detectLegacyCard(card map[string]json.RawMessage, legacyEnabled, legacyPath bool) (bool, error) {
	if legacyPath {
		if !legacyEnabled {
			return false, fmt.Errorf("%w: legacy Agent Cards are disabled", errInvalidAgentCard)
		}
		return true, nil
	}
	if _, ok := card["supportedInterfaces"]; ok {
		return false, nil
	}
	if _, ok := card["url"]; ok && legacyEnabled {
		return true, nil
	}
	return false, nil
}

func validateAndRewriteV1Card(card map[string]json.RawMessage, externalURL string, rewrite, exposeOriginalEndpoints bool) error {
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
		if err := validateEndpointURL(endpoint, exposeOriginalEndpoints); err != nil {
			return fmt.Errorf("supportedInterfaces[%d]: %w", i, err)
		}
		binding, err := requiredString(iface, "protocolBinding")
		if err != nil || !isSupportedBinding(binding) {
			return fmt.Errorf("%w: supportedInterfaces[%d] uses unsupported protocolBinding", errInvalidAgentCard, i)
		}
		if _, err := requiredString(iface, "protocolVersion"); err != nil {
			return fmt.Errorf("%w: supportedInterfaces[%d].protocolVersion is required", errInvalidAgentCard, i)
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

func validateAndRewriteLegacyCard(card map[string]json.RawMessage, externalURL string, rewrite, exposeOriginalEndpoints bool) error {
	endpoint, err := requiredString(card, "url")
	if err != nil {
		return fmt.Errorf("%w: url is required", errInvalidAgentCard)
	}
	if err := validateEndpointURL(endpoint, exposeOriginalEndpoints); err != nil {
		return err
	}
	if raw, ok := card["preferredTransport"]; ok {
		var transport string
		if json.Unmarshal(raw, &transport) != nil || !isSupportedBinding(transport) {
			return fmt.Errorf("%w: unsupported preferredTransport", errInvalidAgentCard)
		}
	}
	if raw, ok := card["additionalInterfaces"]; ok {
		var interfaces []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &interfaces); err != nil {
			return fmt.Errorf("%w: additionalInterfaces must be an array", errInvalidAgentCard)
		}
		for i, iface := range interfaces {
			endpoint, err := requiredString(iface, "url")
			if err != nil {
				return fmt.Errorf("%w: additionalInterfaces[%d].url is required", errInvalidAgentCard, i)
			}
			if err := validateEndpointURL(endpoint, exposeOriginalEndpoints); err != nil {
				return fmt.Errorf("additionalInterfaces[%d]: %w", i, err)
			}
			transport, err := requiredString(iface, "transport")
			if err != nil || !isSupportedBinding(transport) {
				return fmt.Errorf("%w: additionalInterfaces[%d] uses unsupported transport", errInvalidAgentCard, i)
			}
			if rewrite {
				iface["url"], _ = json.Marshal(externalURL)
			}
		}
		if rewrite {
			card["additionalInterfaces"], _ = json.Marshal(interfaces)
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

func isSupportedBinding(binding string) bool {
	normalized := strings.NewReplacer("-", "", "+", "", "_", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(binding)))
	return normalized == "JSONRPC" || normalized == "GRPC" || normalized == "HTTPJSON"
}

func validateHTTPSURL(raw string) error {
	return validateEndpointURL(raw, true)
}

func validateEndpointURL(raw string, requirePublicHTTPS bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Hostname() == "" || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return fmt.Errorf("%w: endpoint must be an absolute HTTP or HTTPS URL", errUnsafeEndpoint)
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("%w: credentials and fragments are forbidden", errUnsafeEndpoint)
	}
	if err := validatePort(parsed.Port()); err != nil {
		return err
	}
	if !requirePublicHTTPS {
		return nil
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("%w: exposed endpoint must use HTTPS", errUnsafeEndpoint)
	}
	return validatePublicHost(parsed.Hostname())
}

func validatePublicHost(host string) error {
	normalized := strings.TrimSuffix(strings.ToLower(host), ".")
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return fmt.Errorf("%w: loopback host is forbidden", errUnsafeEndpoint)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsMulticast() {
			return fmt.Errorf("%w: loopback, link-local, private, multicast, and unspecified addresses are forbidden", errUnsafeEndpoint)
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
	scheme := ctx.scheme()
	authority := ctx.host()
	candidate := strings.ToLower(scheme) + "://" + authority
	if err := validateHTTPSURL(candidate); err != nil {
		return "", fmt.Errorf("request-visible gateway endpoint: %w", err)
	}
	parsedCandidate, err := url.Parse(candidate)
	if err != nil || parsedCandidate.EscapedPath() != "" || parsedCandidate.RawQuery != "" || parsedCandidate.Fragment != "" {
		return "", fmt.Errorf("request-visible gateway authority is malformed")
	}
	routePrefix := config.Agent.ExternalPath
	if routePrefix == "" {
		routePrefix = stripAgentCardSuffix(requestPath, config.AgentCard.Path)
	}
	if routePrefix == "/" {
		routePrefix = ""
	}
	return strings.TrimRight(candidate, "/") + routePrefix, nil
}

func validateExternalPath(externalPath string) error {
	if externalPath == "" {
		return nil
	}
	parsed, err := url.Parse(externalPath)
	if err != nil || !strings.HasPrefix(externalPath, "/") || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("agent.externalPath must be an absolute path without query or fragment")
	}
	return nil
}

type requestContext interface {
	scheme() string
	host() string
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

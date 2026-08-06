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

package protocol

import (
	"mime"
	"net"
	"net/url"
	"strings"
)

const (
	HeaderProtocolVersion = "MCP-Protocol-Version"
	HeaderMethod          = "Mcp-Method"
	HeaderName            = "Mcp-Name"

	// ModernMaxBodyBytes bounds a modern single-message exchange without
	// changing the existing legacy buffer limit.
	ModernMaxBodyBytes uint32 = 1024 * 1024
)

// Transport contains only protocol-relevant request metadata. It intentionally
// excludes authorization, cookies, request IDs, and all runtime session state.
type Transport struct {
	Method                string
	Authority             string
	ContentType           string
	Accept                string
	Origin                string
	ProtocolVersion       string
	MCPMethod             string
	MCPName               string
	AmbiguousHeader       bool
	HasProtocolVersion    bool
	HasMCPMethod          bool
	HasMCPName            bool
	AmbiguousModernHeader bool
}

// NewTransport captures relevant headers without retaining credentials.
// Multiple values for a protocol-sensitive header are rejected later rather
// than guessed or combined.
func NewTransport(method, authority string, headers [][2]string) Transport {
	transport := Transport{Method: method, Authority: authority}
	seen := make(map[string]bool)
	for _, header := range headers {
		name := strings.ToLower(header[0])
		var target *string
		switch name {
		case "content-type":
			target = &transport.ContentType
		case "accept":
			target = &transport.Accept
		case "origin":
			target = &transport.Origin
		case strings.ToLower(HeaderProtocolVersion):
			target = &transport.ProtocolVersion
			transport.HasProtocolVersion = true
		case strings.ToLower(HeaderMethod):
			target = &transport.MCPMethod
			transport.HasMCPMethod = true
		case strings.ToLower(HeaderName):
			target = &transport.MCPName
			transport.HasMCPName = true
		default:
			continue
		}
		if seen[name] {
			transport.AmbiguousHeader = true
			if name == strings.ToLower(HeaderProtocolVersion) || name == strings.ToLower(HeaderMethod) || name == strings.ToLower(HeaderName) {
				transport.AmbiguousModernHeader = true
			}
			continue
		}
		seen[name] = true
		*target = strings.TrimSpace(header[1])
	}
	return transport
}

// HasModernIdentityHeaders reports whether a request carries any header that
// belongs exclusively to the modern profile. Incomplete modern identity must
// never fall back to legacy dispatch.
func HasModernIdentityHeaders(transport Transport) bool {
	return transport.HasMCPMethod || transport.HasMCPName ||
		transport.MCPMethod != "" || transport.MCPName != "" ||
		transport.AmbiguousModernHeader ||
		(transport.HasProtocolVersion && transport.ProtocolVersion == "") ||
		(transport.ProtocolVersion != "" && !IsLegacyVersion(Version(transport.ProtocolVersion)))
}

// ValidateModernTransport performs checks which do not depend on the JSON-RPC
// envelope and therefore can run before request-body processing.
func ValidateModernTransport(transport Transport) *Error {
	if transport.AmbiguousHeader {
		return HeaderMismatch()
	}
	if transport.Method != "POST" {
		return MethodNotAllowed()
	}
	mediaType, _, err := mime.ParseMediaType(transport.ContentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return UnsupportedMediaType()
	}
	if !accepts(transport.Accept, "application/json") || !accepts(transport.Accept, "text/event-stream") {
		return NotAcceptable()
	}
	if !trustedOrigin(transport.Origin, transport.Authority) {
		return UntrustedOrigin()
	}
	return nil
}

func accepts(value, required string) bool {
	for item := range strings.SplitSeq(value, ",") {
		mediaType, parameters, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil {
			continue
		}
		if quality, ok := parameters["q"]; ok && quality == "0" {
			continue
		}
		if strings.EqualFold(mediaType, required) {
			return true
		}
	}
	return false
}

func trustedOrigin(origin, authority string) bool {
	if origin == "" {
		return true
	}
	if strings.ContainsAny(origin, ", ") || authority == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	originHost, originPort, ok := normalizedHostPort(parsed.Host, parsed.Scheme)
	if !ok {
		return false
	}
	authorityHost, authorityPort, ok := normalizedHostPort(authority, parsed.Scheme)
	return ok && strings.EqualFold(originHost, authorityHost) && originPort == authorityPort
}

func normalizedHostPort(value, scheme string) (string, string, bool) {
	host := value
	port := ""
	if parsedHost, parsedPort, err := net.SplitHostPort(value); err == nil {
		host, port = parsedHost, parsedPort
	} else if strings.Count(value, ":") > 1 && !strings.HasPrefix(value, "[") {
		return "", "", false
	}
	if host == "" {
		return "", "", false
	}
	if port == "" {
		switch scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return strings.Trim(host, "[]"), port, true
}

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
	"bytes"
	"encoding/json"
	"sync"
)

// Implementation identifies a modern MCP client when it is provided.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Metadata is required on every modern request. ClientInfo is optional.
type Metadata struct {
	ProtocolVersion    Version
	ClientCapabilities map[string]json.RawMessage
	ClientInfo         *Implementation
}

// Cancellation is scoped to one modern HTTP exchange. It has no protocol
// session identity and is safe to signal more than once.
type Cancellation struct {
	once sync.Once
	done chan struct{}
}

func newCancellation() *Cancellation {
	return &Cancellation{done: make(chan struct{})}
}

func (c *Cancellation) Done() <-chan struct{} {
	return c.done
}

func (c *Cancellation) Cancel() {
	if c == nil {
		return
	}
	c.once.Do(func() { close(c.done) })
}

// RequestContext is the stable pre-dispatch contract shared by modern
// handlers. It deliberately has no protocol session or resumability fields.
type RequestContext struct {
	Era          Era
	Version      Version
	Transport    Transport
	Envelope     Envelope
	Metadata     Metadata
	Cancellation *Cancellation
}

func (c *RequestContext) Cancel() {
	if c != nil {
		c.Cancellation.Cancel()
	}
}

// SemanticResult is the profile-independent result boundary for successor
// handlers. Result shaping remains outside this PROCESS.
type SemanticResult struct {
	Value      any
	ResultType string
	Meta       map[string]any
}

// PrepareRequest classifies a request exactly once. Legacy exchanges are
// passed through unchanged. Modern exchanges complete all transport,
// envelope, identity, lifecycle, and method validation before being returned.
func PrepareRequest(transport Transport, body []byte, methodAvailable func(string) bool) (*RequestContext, *Error) {
	era := classify(transport.ProtocolVersion, body)
	if era == EraLegacy {
		return &RequestContext{
			Era:       EraLegacy,
			Version:   Version(transport.ProtocolVersion),
			Transport: transport,
			Envelope:  Envelope{Raw: bytes.Clone(body)},
		}, nil
	}
	if transport.ProtocolVersion != "" &&
		!IsModernVersion(Version(transport.ProtocolVersion)) &&
		!IsLegacyVersion(Version(transport.ProtocolVersion)) {
		return nil, UnsupportedVersion()
	}
	if protocolError := ValidateModernTransport(transport); protocolError != nil {
		return nil, protocolError
	}
	envelope, protocolError := decodeEnvelope(body)
	if protocolError != nil {
		return nil, protocolError
	}
	request := &RequestContext{
		Era:       EraModern,
		Version:   Version20260728,
		Transport: transport,
		Envelope:  envelope,
	}
	metadata, protocolError := decodeMetadata(envelope.Params)
	if protocolError != nil {
		return request, protocolError
	}
	request.Metadata = metadata
	if !IsModernVersion(metadata.ProtocolVersion) {
		return request, UnsupportedVersion()
	}
	if transport.ProtocolVersion == "" || transport.ProtocolVersion != string(metadata.ProtocolVersion) {
		return request, HeaderMismatch()
	}
	if transport.MCPMethod == "" || transport.MCPMethod != envelope.Method {
		return request, HeaderMismatch()
	}
	if protocolError := validateNameHeader(transport.MCPName, envelope); protocolError != nil {
		return request, protocolError
	}
	if !isCurrentModernMethod(envelope.Method) || methodAvailable == nil || !methodAvailable(envelope.Method) {
		return request, MethodNotFound()
	}
	request.Cancellation = newCancellation()
	return request, nil
}

func classify(headerVersion string, body []byte) Era {
	var envelope struct {
		Params struct {
			Meta map[string]json.RawMessage `json:"_meta"`
		} `json:"params"`
	}
	if json.Unmarshal(body, &envelope) == nil {
		if _, ok := envelope.Params.Meta[MetaProtocolVersion]; ok {
			return EraModern
		}
		if _, ok := envelope.Params.Meta[MetaClientCapabilities]; ok {
			return EraModern
		}
		if _, ok := envelope.Params.Meta[MetaClientInfo]; ok {
			return EraModern
		}
	}
	if headerVersion != "" && !IsLegacyVersion(Version(headerVersion)) {
		return EraModern
	}
	return EraLegacy
}

func decodeMetadata(params json.RawMessage) (Metadata, *Error) {
	if len(params) == 0 {
		return Metadata{}, InvalidParams("modern MCP params._meta is required")
	}
	var decoded struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if err := json.Unmarshal(params, &decoded); err != nil || decoded.Meta == nil {
		return Metadata{}, InvalidParams("modern MCP params._meta is required")
	}
	var version string
	if err := json.Unmarshal(decoded.Meta[MetaProtocolVersion], &version); err != nil || version == "" {
		return Metadata{}, InvalidParams("modern MCP protocolVersion metadata is required")
	}
	capabilitiesRaw, ok := decoded.Meta[MetaClientCapabilities]
	if !ok || !isJSONObject(capabilitiesRaw) {
		return Metadata{}, InvalidParams("modern MCP clientCapabilities metadata is required")
	}
	capabilities := make(map[string]json.RawMessage)
	if err := json.Unmarshal(capabilitiesRaw, &capabilities); err != nil {
		return Metadata{}, InvalidParams("modern MCP clientCapabilities metadata is invalid")
	}
	metadata := Metadata{
		ProtocolVersion:    Version(version),
		ClientCapabilities: capabilities,
	}
	if clientInfoRaw, ok := decoded.Meta[MetaClientInfo]; ok {
		if !isJSONObject(clientInfoRaw) {
			return Metadata{}, InvalidParams("modern MCP clientInfo metadata is invalid")
		}
		var clientInfo Implementation
		if err := json.Unmarshal(clientInfoRaw, &clientInfo); err != nil || clientInfo.Name == "" || clientInfo.Version == "" {
			return Metadata{}, InvalidParams("modern MCP clientInfo metadata is invalid")
		}
		metadata.ClientInfo = &clientInfo
	}
	return metadata, nil
}

func validateNameHeader(nameHeader string, envelope Envelope) *Error {
	if envelope.Method != "tools/call" {
		if nameHeader != "" {
			return HeaderMismatch()
		}
		return nil
	}
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(envelope.Params, &params); err != nil || params.Name == "" {
		return InvalidParams("tools/call params.name is required")
	}
	if nameHeader == "" || nameHeader != params.Name {
		return HeaderMismatch()
	}
	return nil
}

func isCurrentModernMethod(method string) bool {
	switch method {
	case "server/discover", "tools/list", "tools/call":
		return true
	default:
		return false
	}
}

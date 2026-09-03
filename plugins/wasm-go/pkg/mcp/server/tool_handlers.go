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
package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/protocol"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

type directToolSchemaState uint8

const (
	directToolSchemaValidated directToolSchemaState = iota
	directToolSchemaValidationUnavailable
	directToolSchemaExplicitLegacyOnly
)

type directToolEntry struct {
	name         string
	description  string
	inputSchema  map[string]any
	validator    *compiledInputSchema
	schemaState  directToolSchemaState
	serializable bool
	tool         Tool
}

// directToolSnapshot binds discovery and invocation to one analyzed tool
// generation. It is immutable after configuration publication.
type directToolSnapshot struct {
	ordered    []directToolEntry
	byName     map[string]directToolEntry
	legacyOnly map[string]string
	degraded   []directToolDiagnostic
	unlistable []string
}

type directToolDiagnostic struct {
	name   string
	reason schemaDiagnosticReason
}

type capturedInputSchemaTool interface {
	capturedInputSchema() (map[string]any, bool)
}

func compileDirectToolSnapshot(server Server) directToolSnapshot {
	entries := snapshotTools(server)
	snapshot := directToolSnapshot{
		ordered:    make([]directToolEntry, 0, len(entries)),
		byName:     make(map[string]directToolEntry, len(entries)),
		legacyOnly: make(map[string]string),
	}
	for _, entry := range entries {
		var descriptor map[string]any
		var serializable bool
		var validator *compiledInputSchema
		var preparationErr error
		if captured, ok := entry.tool.(capturedInputSchemaTool); ok {
			descriptor, serializable = captured.capturedInputSchema()
			if serializable {
				descriptor, serializable, validator, preparationErr = prepareToolInputSchema(descriptor)
			} else {
				preparationErr = schemaCompileError(schemaDiagnosticSerializationFailure, "input schema is not JSON-compatible")
			}
		} else {
			descriptor, serializable, validator, preparationErr = prepareToolInputSchema(entry.tool.InputSchema())
		}
		if compatibility, ok := entry.tool.(legacySchemaCompatibleTool); ok && compatibility.legacyOnlyInputSchema() {
			reason := "configured as legacy-only"
			if preparationErr != nil {
				reason = preparationErr.Error()
			}
			snapshot.legacyOnly[entry.name] = reason
			explicit := directToolEntry{
				name:         entry.name,
				description:  entry.tool.Description(),
				inputSchema:  descriptor,
				validator:    validator,
				schemaState:  directToolSchemaExplicitLegacyOnly,
				serializable: serializable,
				tool:         entry.tool,
			}
			snapshot.ordered = append(snapshot.ordered, explicit)
			snapshot.byName[entry.name] = explicit
			continue
		}
		state := directToolSchemaValidated
		var reason schemaDiagnosticReason
		if preparationErr != nil {
			state = directToolSchemaValidationUnavailable
			reason = schemaPreparationDiagnosticReason(preparationErr)
		}
		validated := directToolEntry{
			name:         entry.name,
			description:  entry.tool.Description(),
			inputSchema:  descriptor,
			validator:    validator,
			schemaState:  state,
			serializable: serializable,
			tool:         entry.tool,
		}
		snapshot.ordered = append(snapshot.ordered, validated)
		snapshot.byName[entry.name] = validated
		if state == directToolSchemaValidationUnavailable {
			snapshot.degraded = append(snapshot.degraded, directToolDiagnostic{name: entry.name, reason: reason})
			if !serializable {
				snapshot.unlistable = append(snapshot.unlistable, entry.name)
			}
		}
	}
	return snapshot
}

const (
	maxSchemaDiagnosticToolNames     = 8
	maxSchemaDiagnosticToolNameRunes = 128
)

func boundedSchemaDiagnosticToolName(name string) string {
	runes := []rune(name)
	if len(runes) <= maxSchemaDiagnosticToolNameRunes {
		return name
	}
	return string(runes[:maxSchemaDiagnosticToolNameRunes]) + "..."
}

func (snapshot directToolSnapshot) degradedSummary() string {
	if len(snapshot.degraded) == 0 {
		return ""
	}
	reasons := make(map[string]struct{})
	names := make([]string, 0, min(len(snapshot.degraded), maxSchemaDiagnosticToolNames))
	for i, diagnostic := range snapshot.degraded {
		reasons[diagnostic.reason.String()] = struct{}{}
		if i < maxSchemaDiagnosticToolNames {
			names = append(names, boundedSchemaDiagnosticToolName(diagnostic.name))
		}
	}
	reasonNames := make([]string, 0, len(reasons))
	for reason := range reasons {
		reasonNames = append(reasonNames, reason)
	}
	sort.Strings(reasonNames)
	suffix := ""
	if omitted := len(snapshot.degraded) - len(names); omitted > 0 {
		suffix = fmt.Sprintf(" (+%d omitted)", omitted)
	}
	return fmt.Sprintf("%d tool(s), reasons=%s, tools=%s%s", len(snapshot.degraded), strings.Join(reasonNames, ","), strings.Join(names, ","), suffix)
}

func (snapshot directToolSnapshot) unlistableSummary() string {
	if len(snapshot.unlistable) == 0 {
		return ""
	}
	names := make([]string, 0, min(len(snapshot.unlistable), maxSchemaDiagnosticToolNames))
	for i, name := range snapshot.unlistable {
		if i == maxSchemaDiagnosticToolNames {
			break
		}
		names = append(names, boundedSchemaDiagnosticToolName(name))
	}
	suffix := ""
	if omitted := len(snapshot.unlistable) - len(names); omitted > 0 {
		suffix = fmt.Sprintf(" (+%d omitted)", omitted)
	}
	return fmt.Sprintf("%d tool(s), tools=%s%s", len(snapshot.unlistable), strings.Join(names, ","), suffix)
}

type legacySchemaCompatibleTool interface {
	legacyOnlyInputSchema() bool
}

func (snapshot directToolSnapshot) buildModernToolList(effectiveAllowTools *map[string]struct{}) []map[string]any {
	tools := make([]map[string]any, 0, len(snapshot.ordered))
	for _, entry := range snapshot.ordered {
		if entry.schemaState == directToolSchemaExplicitLegacyOnly || !entry.serializable {
			continue
		}
		if effectiveAllowTools != nil {
			if _, allowed := (*effectiveAllowTools)[entry.name]; !allowed {
				continue
			}
		}
		tool := map[string]any{
			"name":        entry.name,
			"description": entry.description,
			"inputSchema": entry.inputSchema,
		}
		tools = append(tools, tool)
	}
	return tools
}

func installDirectToolResultAdapter(ctx wrapper.HttpContext, serverName string) {
	request, modern := ModernRequestContext(ctx)
	if !modern {
		utils.SetMCPResultAdapter(ctx, nil)
		return
	}
	utils.SetMCPResultAdapter(ctx, func(result map[string]any) map[string]any {
		return ShapeResult(request, serverName, protocol.SemanticResult{
			Value:      result,
			ResultType: resultTypeComplete,
		})
	})
}

func sendToolExecutionError(ctx wrapper.HttpContext, err error, debugInfo string) {
	utils.OnMCPToolCallError(ctx, err, debugInfo)
}

func sendSchemaValidationUnavailable(ctx wrapper.HttpContext) {
	request, modern := ModernRequestContext(ctx)
	if !modern {
		return
	}
	utils.SendProtocolError(
		ctx,
		request.Envelope.ID,
		protocol.SchemaValidationUnavailable(),
		"schema_validation_unavailable",
	)
}

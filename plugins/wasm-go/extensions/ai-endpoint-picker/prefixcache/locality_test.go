package prefixcache

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func extractForTest(t *testing.T, body string) *Locality {
	return extractForTestWithToolMode(t, body, DefaultToolMode)
}

func extractForTestWithToolMode(t *testing.T, body string, toolMode ToolMode) *Locality {
	t.Helper()
	locality, supported, err := ExtractWithToolMode([]byte(body), toolMode)
	if err != nil || !supported {
		t.Fatalf("extract failed: supported=%v err=%v", supported, err)
	}
	return locality
}

func TestChatUsesWholeCanonicalMessageSegments(t *testing.T) {
	base := extractForTest(t, `{"model":"m","messages":[{"role":"user","name":"alice","content":"hello","tool_calls":[{"id":"1","type":"function"}]}]}`)
	if got := len(base.Chains[0]); got != 1 {
		t.Fatalf("small complete message produced %d blocks, want 1", got)
	}
	reordered := extractForTest(t, `{"messages":[{"tool_calls":[{"type":"function","id":"1"}],"content":"hello","name":"alice","role":"user"}],"model":"m"}`)
	if base.Chains[0][0].Hash != reordered.Chains[0][0].Hash {
		t.Fatal("canonical message hash changed with JSON field order")
	}
	changedName := extractForTest(t, `{"model":"m","messages":[{"role":"user","name":"bob","content":"hello","tool_calls":[{"id":"1","type":"function"}]}]}`)
	if base.Chains[0][0].Hash == changedName.Chains[0][0].Hash {
		t.Fatal("prompt-relevant message fields must affect the hash")
	}
}

func TestChatToolsAreSeparateOrderedCanonicalSegment(t *testing.T) {
	first := extractForTest(t, `{"model":"m","tools":[{"type":"function","function":{"name":"lookup","parameters":{"b":2,"a":1}}}],"messages":[{"role":"user","content":"hello"}]}`)
	second := extractForTest(t, `{"messages":[{"content":"hello","role":"user"}],"tools":[{"function":{"parameters":{"a":1,"b":2},"name":"lookup"},"type":"function"}],"model":"m"}`)
	if len(first.Chains[0]) != 2 {
		t.Fatalf("tools plus message produced %d blocks, want 2", len(first.Chains[0]))
	}
	for index := range first.Chains[0] {
		if first.Chains[0][index].Hash != second.Chains[0][index].Hash {
			t.Fatalf("canonical block %d differs after field reorder", index)
		}
	}
}

func TestToolModesControlPrefixPrecision(t *testing.T) {
	base := `{"model":"m","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"string"}}}],"messages":[{"role":"user","content":"hello"}]}`
	differentIdentity := `{"model":"m","tools":[{"type":"function","function":{"name":"search","parameters":{"type":"string"}}}],"messages":[{"role":"user","content":"hello"}]}`
	differentType := `{"model":"m","tools":[{"type":"custom","function":{"name":"lookup","parameters":{"type":"string"}}}],"messages":[{"role":"user","content":"hello"}]}`
	differentSchema := `{"model":"m","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"number"}}}],"messages":[{"role":"user","content":"hello"}]}`
	reordered := `{"messages":[{"content":"hello","role":"user"}],"tools":[{"function":{"parameters":{"type":"string"},"name":"lookup"},"type":"function"}],"model":"m"}`

	noneBase := extractForTestWithToolMode(t, base, ToolModeNone)
	noneDifferent := extractForTestWithToolMode(t, differentIdentity, ToolModeNone)
	if noneBase.Chains[0][0].Hash != noneDifferent.Chains[0][0].Hash {
		t.Fatal("none mode included tools in the prefix")
	}

	identityBase := extractForTestWithToolMode(t, base, ToolModeIdentity)
	identityChanged := extractForTestWithToolMode(t, differentIdentity, ToolModeIdentity)
	identityType := extractForTestWithToolMode(t, differentType, ToolModeIdentity)
	identitySchema := extractForTestWithToolMode(t, differentSchema, ToolModeIdentity)
	identityReordered := extractForTestWithToolMode(t, reordered, ToolModeIdentity)
	if identityBase.Chains[0][0].Hash == identityChanged.Chains[0][0].Hash || identityBase.Chains[0][0].Hash == identityType.Chains[0][0].Hash {
		t.Fatal("identity mode did not distinguish tool type/name")
	}
	if identityBase.Chains[0][0].Hash != identitySchema.Chains[0][0].Hash || identityBase.Chains[0][0].Hash != identityReordered.Chains[0][0].Hash {
		t.Fatal("identity mode included schema details or depended on JSON field order")
	}

	fullBase := extractForTestWithToolMode(t, base, ToolModeFull)
	fullSchema := extractForTestWithToolMode(t, differentSchema, ToolModeFull)
	fullReordered := extractForTestWithToolMode(t, reordered, ToolModeFull)
	if fullBase.Chains[0][0].Hash == fullSchema.Chains[0][0].Hash {
		t.Fatal("full mode did not distinguish schema semantics")
	}
	if fullBase.Chains[0][0].Hash != fullReordered.Chains[0][0].Hash {
		t.Fatal("full mode depended on JSON field order")
	}
}

func TestToolIdentityBudgetsPreserveBoundedPrefix(t *testing.T) {
	tool := func(name string) string {
		return `{"type":"function","function":{"name":"` + name + `","description":"` + strings.Repeat("x", 1<<16) + `"}}`
	}
	var capped strings.Builder
	capped.WriteString(`{"model":"m","tools":[`)
	for index := 0; index < MaxTools; index++ {
		if index > 0 {
			capped.WriteByte(',')
		}
		capped.WriteString(tool("stable"))
	}
	capped.WriteString(`],"messages":[]}`)
	extended := strings.TrimSuffix(capped.String(), `],"messages":[]}`) + `,` + tool("ignored") + `],"messages":[]}`
	limited := extractForTestWithToolMode(t, capped.String(), ToolModeIdentity)
	ignoredSuffix := extractForTestWithToolMode(t, extended, ToolModeIdentity)
	if len(limited.Chains[0]) == 0 || limited.Chains[0][len(limited.Chains[0])-1].Hash != ignoredSuffix.Chains[0][len(ignoredSuffix.Chains[0])-1].Hash {
		t.Fatal("MaxTools suffix changed the retained identity prefix")
	}

	first := `{"type":"function","function":{"name":"retained"}}`
	oversized := tool(strings.Repeat("n", MaxToolIdentityBytes))
	left := `{"model":"m","tools":[` + first + `,` + oversized + `,{"type":"function","function":{"name":"left"}}],"messages":[]}`
	right := `{"model":"m","tools":[` + first + `,` + oversized + `,{"type":"function","function":{"name":"right"}}],"messages":[]}`
	leftLocality := extractForTestWithToolMode(t, left, ToolModeIdentity)
	rightLocality := extractForTestWithToolMode(t, right, ToolModeIdentity)
	if len(leftLocality.Chains[0]) == 0 || leftLocality.Chains[0][0].Hash != rightLocality.Chains[0][0].Hash {
		t.Fatal("identity byte cap did not preserve the already produced prefix")
	}
}

func TestToolIdentitySkipsHugeSchemaCanonicalization(t *testing.T) {
	small := `{"model":"m","tools":[{"type":"function","function":{"name":"lookup","parameters":{}}}],"messages":[]}`
	huge := toolSchemaBody(maxCanonicalNodes + 1)
	extended := toolSchemaBody(maxCanonicalNodes + 100_000)
	smallIdentity := extractForTestWithToolMode(t, small, ToolModeIdentity)
	hugeIdentity := extractForTestWithToolMode(t, string(huge), ToolModeIdentity)
	extendedIdentity := extractForTestWithToolMode(t, string(extended), ToolModeIdentity)
	if smallIdentity.Chains[0][0].Hash != hugeIdentity.Chains[0][0].Hash || smallIdentity.Chains[0][0].Hash != extendedIdentity.Chains[0][0].Hash {
		t.Fatal("identity mode included the huge schema")
	}
	hugeAllocs := testing.AllocsPerRun(10, func() { _, _, _ = ExtractWithToolMode(huge, ToolModeIdentity) })
	extendedAllocs := testing.AllocsPerRun(10, func() { _, _, _ = ExtractWithToolMode(extended, ToolModeIdentity) })
	if extendedAllocs > hugeAllocs+8 {
		t.Fatalf("identity allocations scaled with schema nodes: huge=%v extended=%v", hugeAllocs, extendedAllocs)
	}
	if locality, supported, err := ExtractWithToolMode(huge, ToolModeFull); err != nil || supported || locality != nil {
		t.Fatalf("full mode should enforce canonical node budget: locality=%+v supported=%v err=%v", locality, supported, err)
	}
}

func TestLongMessageSplitsAtBoundedSemanticSize(t *testing.T) {
	content := strings.Repeat("a", MaxSegmentTokens*4*2)
	body, _ := json.Marshal(map[string]any{
		"model":    "m",
		"messages": []any{map[string]any{"role": "user", "content": content}},
	})
	locality := extractForTest(t, string(body))
	if got := len(locality.Chains[0]); got != 3 {
		t.Fatalf("long canonical message produced %d blocks, want 3", got)
	}
	for _, block := range locality.Chains[0] {
		if block.EstimatedTokens <= 0 || block.EstimatedTokens > MaxSegmentTokens {
			t.Fatalf("unbounded segment length %d", block.EstimatedTokens)
		}
	}
}

func TestLongMessageMetadataChangesInvalidateFirstSegment(t *testing.T) {
	content := strings.Repeat("a", MaxSegmentTokens*4*3)
	tests := []struct {
		name  string
		first map[string]any
		other map[string]any
	}{
		{
			name:  "role",
			first: map[string]any{"role": "user", "content": content},
			other: map[string]any{"role": "system", "content": content},
		},
		{
			name:  "name",
			first: map[string]any{"role": "user", "name": "alice", "content": content},
			other: map[string]any{"role": "user", "name": "bob", "content": content},
		},
		{
			name: "tool_calls",
			first: map[string]any{
				"role": "assistant", "content": content,
				"tool_calls": []any{map[string]any{"id": "one", "type": "function"}},
			},
			other: map[string]any{
				"role": "assistant", "content": content,
				"tool_calls": []any{map[string]any{"id": "two", "type": "function"}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			firstBody, _ := json.Marshal(map[string]any{"model": "m", "messages": []any{test.first}})
			otherBody, _ := json.Marshal(map[string]any{"model": "m", "messages": []any{test.other}})
			first := extractForTest(t, string(firstBody))
			other := extractForTest(t, string(otherBody))
			if len(first.Chains[0]) < 3 || len(first.Chains[0]) != len(other.Chains[0]) {
				t.Fatalf("unexpected long-message chains: %d vs %d", len(first.Chains[0]), len(other.Chains[0]))
			}
			if first.Chains[0][0].Hash == other.Chains[0][0].Hash {
				t.Fatal("message framing metadata did not isolate the first segment")
			}
			index := NewIndex(DefaultCapacity)
			index.Record("a", first.Chains, DefaultBlockSize)
			if score := index.Score("a", other.Chains); score != 0 {
				t.Fatalf("metadata change retained a false prefix score %v", score)
			}
		})
	}
}

func TestLongMessageContentSuffixRetainsEarlierChunks(t *testing.T) {
	prefix := strings.Repeat("a", MaxSegmentTokens*4*2)
	firstBody, _ := json.Marshal(map[string]any{
		"model": "m", "messages": []any{map[string]any{"role": "user", "name": "same", "content": prefix + "first suffix"}},
	})
	otherBody, _ := json.Marshal(map[string]any{
		"model": "m", "messages": []any{map[string]any{"name": "same", "content": prefix + "other suffix", "role": "user"}},
	})
	first := extractForTest(t, string(firstBody))
	other := extractForTest(t, string(otherBody))
	if len(first.Chains[0]) != 3 || len(other.Chains[0]) != 3 {
		t.Fatalf("unexpected suffix chains: %d vs %d", len(first.Chains[0]), len(other.Chains[0]))
	}
	for block := 0; block < 2; block++ {
		if first.Chains[0][block].Hash != other.Chains[0][block].Hash {
			t.Fatalf("unchanged content prefix block %d did not match", block)
		}
	}
	if first.Chains[0][2].Hash == other.Chains[0][2].Hash {
		t.Fatal("changed content suffix retained the final block")
	}
}

func TestCompletionsTextTokenIDsAndMultiplePrompts(t *testing.T) {
	text := extractForTest(t, `{"model":"m","prompt":"hello"}`)
	if len(text.Chains) != 1 || len(text.Chains[0]) != 1 || text.Chains[0][0].EstimatedTokens != 2 {
		t.Fatalf("text completion locality=%+v", text.Chains)
	}
	tokens := extractForTest(t, `{"model":"m","prompt":[1,2,3]}`)
	if len(tokens.Chains) != 1 || tokens.Chains[0][0].EstimatedTokens != 3 {
		t.Fatalf("token completion locality=%+v", tokens.Chains)
	}
	multiple := extractForTest(t, `{"model":"m","prompt":["first","second"]}`)
	if len(multiple.Chains) != 2 || len(multiple.Chains[0]) != 1 || len(multiple.Chains[1]) != 1 {
		t.Fatalf("multiple prompts did not create independent chains: %+v", multiple.Chains)
	}
}

func TestCompletionTokenPromptShapes(t *testing.T) {
	flat := extractForTest(t, `{"model":"m","prompt":[1,2,3]}`)
	if len(flat.Chains) != 1 || totalTokens(flat.Chains) != 3 {
		t.Fatalf("flat token prompt locality=%+v", flat.Chains)
	}

	if locality, supported, err := Extract([]byte(`{"model":"m","prompt":[[1,2],[3,4]]}`)); err != nil || supported || locality != nil {
		t.Fatalf("batched token prompt must be prefix-unsupported: locality=%+v supported=%v err=%v", locality, supported, err)
	}

	invalid := []string{
		`{"model":"m","prompt":[[1,2],3]}`,
		`{"model":"m","prompt":[[1,"two"],[3,4]]}`,
		`{"model":"m","prompt":[1,[2,3]]}`,
	}
	for _, body := range invalid {
		if _, _, err := Extract([]byte(body)); err == nil {
			t.Fatalf("mixed or invalid token prompt succeeded: %s", body)
		}
	}
}

func TestCanonicalJSONDepthLimit(t *testing.T) {
	for _, test := range []struct {
		depth   int
		wantErr bool
	}{
		{depth: 63},
		{depth: 64},
		{depth: 65, wantErr: true},
	} {
		writer := &canonicalBudgetWriter{target: &countingWriter{}, remaining: maxCanonicalNodes}
		err := writeCanonicalJSON(nestedJSONArray(test.depth), writer)
		if got := errors.Is(err, errJSONDepthLimit); got != test.wantErr {
			t.Fatalf("depth %d error=%v, depth-limit=%v want %v", test.depth, err, got, test.wantErr)
		}
		locality, supported, err := ExtractWithToolMode(deepToolsBody(test.depth-1), ToolModeFull)
		if test.wantErr {
			if err != nil || supported || locality != nil {
				t.Fatalf("depth %d extraction must be prefix-unsupported: locality=%+v supported=%v err=%v", test.depth, locality, supported, err)
			}
		} else if err != nil || !supported || locality == nil {
			t.Fatalf("depth %d extraction failed: locality=%+v supported=%v err=%v", test.depth, locality, supported, err)
		}
	}
}

func TestDeepCanonicalInputHasBoundedTraversal(t *testing.T) {
	depth65 := deepToolsBody(65)
	depth5000 := deepToolsBody(5000)
	for _, body := range [][]byte{depth65, depth5000} {
		locality, supported, err := ExtractWithToolMode(body, ToolModeFull)
		if err != nil || supported || locality != nil {
			t.Fatalf("deep tools must be prefix-unsupported: locality=%+v supported=%v err=%v", locality, supported, err)
		}
	}
	baselineAllocs := testing.AllocsPerRun(10, func() { _, _, _ = ExtractWithToolMode(depth65, ToolModeFull) })
	deepAllocs := testing.AllocsPerRun(10, func() { _, _, _ = ExtractWithToolMode(depth5000, ToolModeFull) })
	if deepAllocs > baselineAllocs+4 {
		t.Fatalf("deep traversal allocations grew with depth: depth65=%v depth5000=%v", baselineAllocs, deepAllocs)
	}
}

func nestedJSONArray(depth int) []byte {
	var value strings.Builder
	value.Grow(depth*2 + 1)
	value.WriteString(strings.Repeat("[", depth))
	value.WriteByte('0')
	value.WriteString(strings.Repeat("]", depth))
	return []byte(value.String())
}

func deepToolsBody(depth int) []byte {
	value := nestedJSONArray(depth)
	body := make([]byte, 0, len(value)+43)
	body = append(body, `{"model":"m","tools":`...)
	body = append(body, value...)
	body = append(body, `,"messages":[]}`...)
	return body
}

func TestNamespaceSegmentKindAndOutputIsolation(t *testing.T) {
	base := extractForTest(t, `{"model":"m","cache_salt":"a","prompt":"same","temperature":0}`)
	outputChanged := extractForTest(t, `{"model":"m","cache_salt":"a","prompt":"same","temperature":1,"max_tokens":999}`)
	otherModel := extractForTest(t, `{"model":"other","cache_salt":"a","prompt":"same"}`)
	otherSalt := extractForTest(t, `{"model":"m","cache_salt":"b","prompt":"same"}`)
	ambiguousA := extractForTest(t, `{"model":"ab","cache_salt":"c","prompt":"same"}`)
	ambiguousB := extractForTest(t, `{"model":"a","cache_salt":"bc","prompt":"same"}`)
	chat := extractForTest(t, `{"model":"m","cache_salt":"a","messages":[{"role":"user","content":"same"}]}`)
	baseHash := base.Chains[0][0].Hash
	if baseHash != outputChanged.Chains[0][0].Hash {
		t.Fatal("output-generation fields affected prefix hash")
	}
	if baseHash == otherModel.Chains[0][0].Hash || baseHash == otherSalt.Chains[0][0].Hash || baseHash == chat.Chains[0][0].Hash {
		t.Fatal("namespace or segment kind did not isolate prefix hash")
	}
	if ambiguousA.Chains[0][0].Hash == ambiguousB.Chains[0][0].Hash {
		t.Fatal("model and cache_salt namespace components were not length-delimited")
	}
}

func TestEscapedStringsAndKeysCanonicalizeDeterministically(t *testing.T) {
	plain := extractForTest(t, `{"model":"m","messages":[{"role":"user","content":"hello 世界"}]}`)
	escaped := extractForTest(t, `{"model":"m","messages":[{"r\u006fle":"user","content":"\u0068ello \u4e16\u754c"}]}`)
	if plain.Chains[0][0].Hash != escaped.Chains[0][0].Hash {
		t.Fatal("equivalent JSON string/key escapes produced different semantic hashes")
	}
}

func TestMessageContentRepresentationIsDomainSeparated(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
	}{
		{
			name:  "absent versus empty string",
			left:  `{"model":"m","messages":[{"role":"user"}]}`,
			right: `{"model":"m","messages":[{"role":"user","content":""}]}`,
		},
		{
			name:  "null versus null string",
			left:  `{"model":"m","messages":[{"role":"user","content":null}]}`,
			right: `{"model":"m","messages":[{"role":"user","content":"null"}]}`,
		},
		{
			name:  "structured versus matching plain string",
			left:  `{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`,
			right: `{"model":"m","messages":[{"role":"user","content":"[{\"text\":\"hello\",\"type\":\"text\"}]"}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left := extractForTest(t, test.left)
			right := extractForTest(t, test.right)
			if left.Chains[0][0].Hash == right.Chains[0][0].Hash {
				t.Fatal("different content representations produced the same first hash")
			}
		})
	}

	ordered := extractForTest(t, `{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	reordered := extractForTest(t, `{"messages":[{"content":[{"text":"hello","type":"text"}],"role":"user"}],"model":"m"}`)
	if ordered.Chains[0][0].Hash != reordered.Chains[0][0].Hash {
		t.Fatal("structured content hash changed with JSON field order")
	}
}

func TestPrefixTokenCapDoesNotCreateTinyBlocks(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"model":  "m",
		"prompt": strings.Repeat("a", (MaxPrefixTokens+MaxSegmentTokens)*4),
	})
	locality := extractForTest(t, string(body))
	if got, want := len(locality.Chains[0]), MaxPrefixTokens/MaxSegmentTokens; got != want {
		t.Fatalf("block count=%d want %d", got, want)
	}
	total := 0
	for _, block := range locality.Chains[0] {
		total += block.EstimatedTokens
	}
	if total != MaxPrefixTokens {
		t.Fatalf("estimated tokens=%d want capped %d", total, MaxPrefixTokens)
	}
}

func TestTokenIDPromptIsStreamedAndCapped(t *testing.T) {
	var body strings.Builder
	body.WriteString(`{"model":"m","prompt":[`)
	for token := 0; token < MaxPrefixTokens+MaxSegmentTokens; token++ {
		if token > 0 {
			body.WriteByte(',')
		}
		body.WriteString(strconv.Itoa(token))
	}
	body.WriteString(`]}`)
	locality := extractForTest(t, body.String())
	if got, want := len(locality.Chains[0]), MaxPrefixTokens/MaxSegmentTokens; got != want {
		t.Fatalf("token-ID block count=%d want %d", got, want)
	}
	if got := totalTokens(locality.Chains); got != MaxPrefixTokens {
		t.Fatalf("token-ID count=%d want capped %d", got, MaxPrefixTokens)
	}
}

func TestTokenIDParserIsStrictAndAllocationFree(t *testing.T) {
	tests := []struct {
		raw  string
		want uint32
		ok   bool
	}{
		{raw: "0", ok: true},
		{raw: "4294967295", want: ^uint32(0), ok: true},
		{raw: "4294967296"},
		{raw: "-1"},
		{raw: "1.0"},
		{raw: "01"},
	}
	for _, test := range tests {
		got, ok := parseUint32([]byte(test.raw))
		if got != test.want || ok != test.ok {
			t.Fatalf("parseUint32(%q)=(%d,%v) want (%d,%v)", test.raw, got, ok, test.want, test.ok)
		}
	}
	if allocations := testing.AllocsPerRun(100, func() {
		_, _ = parseUint32([]byte("4294967295"))
	}); allocations != 0 {
		t.Fatalf("parseUint32 allocations=%v want 0", allocations)
	}
}

func TestStructuredContentCapFinishesPartialBlock(t *testing.T) {
	body := repeatedStructuredBody(140, 0, false, true)
	locality, supported, err := Extract(body)
	if err != nil || !supported {
		t.Fatalf("extract: supported=%v err=%v", supported, err)
	}
	if got := totalTokens(locality.Chains); got != MaxPrefixTokens {
		t.Fatalf("token count=%d want %d", got, MaxPrefixTokens)
	}
	chain := locality.Chains[0]
	if got := chain[len(chain)-1].EstimatedTokens; got <= 0 || got >= MaxSegmentTokens {
		t.Fatalf("last partial block tokens=%d want between 1 and %d", got, MaxSegmentTokens-1)
	}
}

func TestMultiMegabyteTextPromptsRemainAllocationBounded(t *testing.T) {
	content := strings.Repeat("a", 4<<20)
	tests := []struct {
		name string
		body []byte
	}{
		{name: "completion", body: mustJSON(t, map[string]any{"model": "m", "prompt": content})},
		{name: "chat", body: mustJSON(t, map[string]any{
			"model": "m", "messages": []any{map[string]any{"role": "user", "content": content}},
		})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			locality, supported, err := Extract(test.body)
			if err != nil || !supported {
				t.Fatalf("extract: supported=%v err=%v", supported, err)
			}
			if got := len(locality.Chains[0]); got != MaxPrefixTokens/MaxSegmentTokens {
				t.Fatalf("semantic segment count=%d want %d", got, MaxPrefixTokens/MaxSegmentTokens)
			}
			if got := totalTokens(locality.Chains); got != MaxPrefixTokens {
				t.Fatalf("token count=%d want %d", got, MaxPrefixTokens)
			}
			if allocations := testing.AllocsPerRun(3, func() {
				_, _, _ = Extract(test.body)
			}); allocations > 100 {
				t.Fatalf("allocations=%v want bounded <=100", allocations)
			}
		})
	}
}

func TestCappedSemanticSuffixesDoNotScaleAllocations(t *testing.T) {
	tests := []struct {
		name            string
		capped          []byte
		extended        []byte
		maxCappedAllocs float64
		toolMode        ToolMode
	}{
		{
			name:            "token IDs",
			capped:          tokenIDBody(MaxPrefixTokens),
			extended:        tokenIDBody(MaxPrefixTokens + 1_000_000),
			maxCappedAllocs: 64,
		},
		{
			name:     "messages",
			capped:   repeatedChatBody(140, 0),
			extended: repeatedChatBody(140, 100_000),
		},
		{
			name:            "tools",
			capped:          repeatedToolsBody(140, 0),
			extended:        repeatedToolsBody(140, 100_000),
			maxCappedAllocs: 10_000,
			toolMode:        ToolModeFull,
		},
		{
			name:     "structured content",
			capped:   repeatedStructuredBody(140, 0, false, false),
			extended: repeatedStructuredBody(140, 100_000, true, false),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toolMode := test.toolMode
			if toolMode == "" {
				toolMode = DefaultToolMode
			}
			for _, body := range [][]byte{test.capped, test.extended} {
				locality, supported, err := ExtractWithToolMode(body, toolMode)
				if err != nil || !supported {
					t.Fatalf("extract: supported=%v err=%v", supported, err)
				}
				if got := totalTokens(locality.Chains); got != MaxPrefixTokens {
					t.Fatalf("token count=%d want capped %d", got, MaxPrefixTokens)
				}
			}
			cappedAllocs := testing.AllocsPerRun(3, func() { _, _, _ = ExtractWithToolMode(test.capped, toolMode) })
			extendedAllocs := testing.AllocsPerRun(3, func() { _, _, _ = ExtractWithToolMode(test.extended, toolMode) })
			t.Logf("allocations capped=%v extended=%v", cappedAllocs, extendedAllocs)
			if test.maxCappedAllocs > 0 && cappedAllocs > test.maxCappedAllocs {
				t.Fatalf("capped allocations=%v want <=%v", cappedAllocs, test.maxCappedAllocs)
			}
			if extendedAllocs > cappedAllocs+8 {
				t.Fatalf("allocations scaled after cap: capped=%v extended=%v", cappedAllocs, extendedAllocs)
			}
		})
	}
}

func TestToolsCanonicalComplexityIsBounded(t *testing.T) {
	capped := toolsObjectBody(maxCanonicalNodes + 1)
	extended := toolsObjectBody(maxCanonicalNodes + 100_000)
	for _, raw := range [][]byte{capped, extended} {
		_, supported, err := ExtractWithToolMode(raw, ToolModeFull)
		if err != nil || supported {
			t.Fatalf("complex tools should make prefix unavailable: supported=%v err=%v", supported, err)
		}
	}
	cappedAllocs := testing.AllocsPerRun(3, func() { _, _, _ = ExtractWithToolMode(capped, ToolModeFull) })
	extendedAllocs := testing.AllocsPerRun(3, func() { _, _, _ = ExtractWithToolMode(extended, ToolModeFull) })
	if cappedAllocs > 70_000 || extendedAllocs > cappedAllocs+8 {
		t.Fatalf("complexity allocations not bounded: capped=%v extended=%v", cappedAllocs, extendedAllocs)
	}
}

func toolsObjectBody(count int) []byte {
	var body strings.Builder
	body.Grow(count*3 + 50)
	body.WriteString(`{"model":"m","tools":[`)
	for index := 0; index < count; index++ {
		if index > 0 {
			body.WriteByte(',')
		}
		body.WriteString(`{}`)
	}
	body.WriteString(`],"messages":[]}`)
	return []byte(body.String())
}

func toolSchemaBody(fields int) []byte {
	var body strings.Builder
	body.Grow(fields*12 + 100)
	body.WriteString(`{"model":"m","tools":[{"type":"function","function":{"name":"lookup","parameters":{`)
	for index := 0; index < fields; index++ {
		if index > 0 {
			body.WriteByte(',')
		}
		body.WriteString(`"k`)
		body.WriteString(strconv.Itoa(index))
		body.WriteString(`":0`)
	}
	body.WriteString(`}}}],"messages":[]}`)
	return []byte(body.String())
}

func tokenIDBody(count int) []byte {
	var body strings.Builder
	body.Grow(count*2 + 30)
	body.WriteString(`{"model":"m","prompt":[`)
	for token := 0; token < count; token++ {
		if token > 0 {
			body.WriteByte(',')
		}
		body.WriteByte(byte('0' + token%10))
	}
	body.WriteString(`]}`)
	return []byte(body.String())
}

func repeatedChatBody(prefixMessages, suffixMessages int) []byte {
	const prefix = `{"role":"user","content":"`
	const suffix = `"}`
	const small = `{"role":"user","content":"x"}`
	var body strings.Builder
	body.Grow(prefixMessages*(len(prefix)+maxSegmentBytes+len(suffix)) + suffixMessages*(len(small)+1) + 40)
	body.WriteString(`{"model":"m","messages":[`)
	for message := 0; message < prefixMessages+suffixMessages; message++ {
		if message > 0 {
			body.WriteByte(',')
		}
		if message < prefixMessages {
			body.WriteString(prefix)
			body.WriteString(strings.Repeat("a", maxSegmentBytes))
			body.WriteString(suffix)
		} else {
			body.WriteString(small)
		}
	}
	body.WriteString(`]}`)
	return []byte(body.String())
}

func repeatedToolsBody(prefixTools, suffixTools int) []byte {
	const prefix = `{"type":"function","function":{"name":"f","description":"`
	const suffix = `"}}`
	const small = `{"type":"function","function":{"name":"f"}}`
	var body strings.Builder
	body.Grow(prefixTools*(len(prefix)+maxSegmentBytes+len(suffix)) + suffixTools*(len(small)+1) + 60)
	body.WriteString(`{"model":"m","tools":[`)
	for tool := 0; tool < prefixTools+suffixTools; tool++ {
		if tool > 0 {
			body.WriteByte(',')
		}
		if tool < prefixTools {
			body.WriteString(prefix)
			body.WriteString(strings.Repeat("a", maxSegmentBytes))
			body.WriteString(suffix)
		} else {
			body.WriteString(small)
		}
	}
	body.WriteString(`],"messages":[]}`)
	return []byte(body.String())
}

func repeatedStructuredBody(prefixParts, suffixParts int, appendUnsupported, withTools bool) []byte {
	content := repeatedStructuredContent(prefixParts, suffixParts, appendUnsupported)
	var body strings.Builder
	body.Grow(len(content) + 100)
	body.WriteString(`{"model":"m"`)
	if withTools {
		body.WriteString(`,"tools":[{}]`)
	}
	body.WriteString(`,"messages":[{"role":"user","content":`)
	body.Write(content)
	body.WriteString(`}]}`)
	return []byte(body.String())
}

func repeatedStructuredContent(prefixParts, suffixParts int, appendUnsupported bool) []byte {
	const prefix = `{"type":"text","text":"`
	const suffix = `"}`
	const small = `{"type":"text","text":"x"}`
	var content strings.Builder
	content.Grow(prefixParts*(len(prefix)+maxSegmentBytes+len(suffix)) + suffixParts*(len(small)+1) + 70)
	content.WriteByte('[')
	for part := 0; part < prefixParts+suffixParts; part++ {
		if part > 0 {
			content.WriteByte(',')
		}
		if part < prefixParts {
			content.WriteString(prefix)
			content.WriteString(strings.Repeat("a", maxSegmentBytes))
			content.WriteString(suffix)
		} else {
			content.WriteString(small)
		}
	}
	if appendUnsupported {
		if prefixParts+suffixParts > 0 {
			content.WriteByte(',')
		}
		content.WriteString(`{"type":"image_url","image_url":{"url":"ignored-after-cap"}}`)
	}
	content.WriteByte(']')
	return []byte(content.String())
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestUnsupportedMultimodalOnlyDisablesPrefix(t *testing.T) {
	tests := []string{
		`{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"describe"},{"type":"image_url","image_url":{"url":"x"}}]}]}`,
		`{"model":"m","messages":[{"role":"assistant","content":null,"audio":{"id":"audio-1"}}]}`,
	}
	for _, body := range tests {
		_, supported, err := Extract([]byte(body))
		if err != nil || supported {
			t.Fatalf("multimodal request should be unavailable without error: supported=%v err=%v", supported, err)
		}
	}

	_, supported, err := Extract([]byte(`{"model":"m","messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function"}]}]}`))
	if err != nil || !supported {
		t.Fatalf("normal tool_calls must remain supported: supported=%v err=%v", supported, err)
	}
}

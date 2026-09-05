# streamxform — streaming request body transform

Converts an OpenAI-style request body to the target provider's format **while the body is still arriving**,
so ai-proxy no longer has to buffer the whole request before protocol conversion. Memory stays independent of
request size; untouched bytes are forwarded verbatim.

## Layers

| Layer | Files | Role |
|---|---|---|
| Scanner + writer | `engine.go`, `writer.go`, `action.go`, `jsonutil.go` | Protocol-agnostic. Walks the JSON byte stream, dispatches every key / array element of an *entered* container to the protocol, and executes the returned action. The writer builds output lazily: a container that never receives a write leaves no trace. |
| Protocols | `proto_claude.go`, `proto_gemini.go`, `proto_qwen.go`, `proto_openai.go`, `proto_openai_variants.go` | Hand-written, one per target format, derived line by line from the existing buffered transforms (`buildClaudeTextGenRequest`, `buildGeminiChatRequest`, `buildQwenTextGenerationRequest`, `defaultTransformRequestBody` …). No rule tables. |
| Guard | `../main.go` (`xformState`), `../provider/streamhooks.go` (`NewStreamPlan`) | Holds the request headers (ActionPause) until a 64KB commit point, applies the header / context side effects the buffered path would have applied, falls back to the buffered path when a shape is unsupported before the commit point, fails the request (500, `ai-proxy.stream_xform_uncoverable`) after it. |

## Actions a protocol can return

| Action | Effect | Buffering |
|---|---|---|
| `Pass` | key + value forwarded verbatim (`As` renames, `Wrap` adds prefix/suffix, `Inner` strips quotes, `At` writes to an outer output level) | none |
| `Skip` | dropped | none |
| `Enter` | descend; children are dispatched too (`Lazy`: omit if empty, `Flat`: no output level, `Lenient`: scalar becomes `Pass`) | none |
| `Probe` | look at the value type first, then decide (`OnStart`) | none |
| `Observe(cap)` | `Pass` plus a copy handed to `OnValue` | bounded |
| `Capture(cap)` | value collected and handed to `OnValue`; nothing written | bounded |
| `Defer(cap)` | key + value held; re-dispatched when the protocol calls `Release` (e.g. `content` arriving before `role`) | bounded |
| `Prefix(cap)` | string: first `cap` bytes handed to `OnPrefix`, which decides how the rest streams (e.g. `data:` image URLs) | window only |
| `Bail` | unsupported → fall back / fail | — |

Small fields (model, role, tools, thinking config …) are captured and rewritten with the same Go structs the
buffered path uses, so `omitempty` and field shapes match byte for byte. Long strings and base64 attachments
never enter memory.

## Correctness

Each protocol is checked against the original implementation in the same package tests:
hand-written cases × chunk sizes 1/7/4096 plus randomized fuzzing (`provider/streamxform_*_test.go`,
`STREAMXFORM_FUZZ_N` scales the fuzz size). `../streaming_request_test.go` drives the whole plugin through
the wasm-go host emulator chunk by chunk (Pause / Continue / fallback / 500 / passthrough).

Known deliberate differences from the buffered path: the buffered path type-checks the whole body against
its structs and returns 500 on any mismatch, the streaming path validates only what it reads; when `stream`
appears after the first 64KB the `Accept` header is not rewritten (providers decide streaming by the body).

## Adding a protocol

1. Read the buffered transform for the provider and list every field it reads and every output field.
2. Write `proto_<name>.go`: dispatch by `t.Depth()` / `t.Last()`, capture small fields, stream long ones,
   put aggregated output in `Tail`. Anything you cannot express → `Bail`.
3. Add a differential test that calls the original builder and compares field by field.
4. Register it in `provider/streamhooks.go` (`NewStreamPlan`), including any header / context side effects
   and the pre-commit requirements (`RequireModelBeforeCommit` / `RequireStreamBeforeCommit`).

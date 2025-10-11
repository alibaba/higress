package pkg

import (
	"strings"
)

type Stage int

const (
	StageInvalid Stage = iota
	StageRequestHeaders
	StageRequestBody
	StageResponseHeaders
	StageResponseBody
)

var (
	OrderedStages = []Stage{
		StageRequestHeaders,
		StageRequestBody,
		StageResponseHeaders,
		StageResponseBody,
	}
	Stage2String = map[Stage]string{
		StageRequestHeaders:  "request_headers",
		StageRequestBody:     "request_body",
		StageResponseHeaders: "response_headers",
		StageResponseBody:    "response_body",
	}
)

type EditorContext interface {
	GetEffectiveCommandSet() *CommandSet
	SetEffectiveCommandSet(cmdSet *CommandSet)
	GetCommandExecutors() []Executor
	SetCommandExecutors(executors []Executor)
	GetCurrentStage() Stage
	SetCurrentStage(stage Stage)

	GetRequestPath() string
	SetRequestPath(path string)
	GetRequestHeader(key string) []string
	GetRequestHeaders() map[string][]string
	SetRequestHeaders(map[string][]string)
	GetRequestQuery(key string) []string
	GetRequestQueries() map[string][]string
	SetRequestQueries(map[string][]string)
	GetResponseHeader(key string) []string
	GetResponseHeaders() map[string][]string
	SetResponseHeaders(map[string][]string)

	GetRefValue(ref *Ref) string
	GetRefValues(ref *Ref) []string
	SetRefValue(ref *Ref, value string)
	SetRefValues(ref *Ref, values []string)
	DeleteRefValues(ref *Ref)

	IsRequestHeadersDirty() bool
	IsRequestQueriesDirty() bool
	IsResponseHeadersDirty() bool
	ResetDirtyFlags()
}

func NewEditorContext() EditorContext {
	return &editorContext{}
}

type editorContext struct {
	effectiveCommandSet *CommandSet
	commandExecutors    []Executor

	currentStage Stage

	requestPath     string
	requestHeaders  map[string][]string
	requestQueries  map[string][]string
	responseHeaders map[string][]string

	requestHeadersDirty  bool
	requestQueriesDirty  bool
	responseHeadersDirty bool
}

func (ctx *editorContext) GetEffectiveCommandSet() *CommandSet {
	return ctx.effectiveCommandSet
}

func (ctx *editorContext) SetEffectiveCommandSet(cmdSet *CommandSet) {
	ctx.effectiveCommandSet = cmdSet
}

func (ctx *editorContext) GetCommandExecutors() []Executor {
	return ctx.commandExecutors
}

func (ctx *editorContext) SetCommandExecutors(executors []Executor) {
	ctx.commandExecutors = executors
}

func (ctx *editorContext) GetCurrentStage() Stage {
	return ctx.currentStage
}

func (ctx *editorContext) SetCurrentStage(stage Stage) {
	ctx.currentStage = stage
}

func (ctx *editorContext) GetRequestPath() string {
	return ctx.requestPath
}

func (ctx *editorContext) SetRequestPath(path string) {
	ctx.requestPath = path
}

func (ctx *editorContext) GetRequestHeader(key string) []string {
	if ctx.requestHeaders == nil {
		return nil
	}
	return ctx.requestHeaders[strings.ToLower(key)]
}

func (ctx *editorContext) GetRequestHeaders() map[string][]string {
	return ctx.requestHeaders
}

func (ctx *editorContext) SetRequestHeaders(headers map[string][]string) {
	ctx.requestHeaders = headers
	ctx.requestHeadersDirty = true
}

func (ctx *editorContext) GetRequestQuery(key string) []string {
	if ctx.requestQueries == nil {
		return nil
	}
	return ctx.requestQueries[key]
}

func (ctx *editorContext) GetRequestQueries() map[string][]string {
	return ctx.requestQueries
}

func (ctx *editorContext) SetRequestQueries(queries map[string][]string) {
	ctx.requestQueries = queries
	ctx.requestQueriesDirty = true
}

func (ctx *editorContext) GetResponseHeader(key string) []string {
	if ctx.responseHeaders == nil {
		return nil
	}
	return ctx.responseHeaders[strings.ToLower(key)]
}

func (ctx *editorContext) GetResponseHeaders() map[string][]string {
	return ctx.responseHeaders
}

func (ctx *editorContext) SetResponseHeaders(headers map[string][]string) {
	ctx.responseHeaders = headers
	ctx.responseHeadersDirty = true
}

func (ctx *editorContext) GetRefValue(ref *Ref) string {
	values := ctx.GetRefValues(ref)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (ctx *editorContext) GetRefValues(ref *Ref) []string {
	if ref == nil {
		return nil
	}
	switch ref.Type {
	case RefTypeRequestHeader:
		return ctx.GetRequestHeader(strings.ToLower(ref.Name))
	case RefTypeRequestQuery:
		return ctx.GetRequestQuery(ref.Name)
	case RefTypeResponseHeader:
		return ctx.GetResponseHeader(strings.ToLower(ref.Name))
	default:
		return nil
	}
}

func (ctx *editorContext) SetRefValue(ref *Ref, value string) {
	if ref == nil {
		return
	}
	ctx.SetRefValues(ref, []string{value})
}

func (ctx *editorContext) SetRefValues(ref *Ref, values []string) {
	if ref == nil {
		return
	}
	switch ref.Type {
	case RefTypeRequestHeader:
		if ctx.requestHeaders == nil {
			ctx.requestHeaders = make(map[string][]string)
		}
		ctx.requestHeaders[strings.ToLower(ref.Name)] = values
		ctx.requestHeadersDirty = true
	case RefTypeRequestQuery:
		if ctx.requestQueries == nil {
			ctx.requestQueries = make(map[string][]string)
		}
		ctx.requestQueries[ref.Name] = values
		ctx.requestQueriesDirty = true
	case RefTypeResponseHeader:
		if ctx.responseHeaders == nil {
			ctx.responseHeaders = make(map[string][]string)
		}
		ctx.responseHeaders[strings.ToLower(ref.Name)] = values
		ctx.responseHeadersDirty = true
	}
}

func (ctx *editorContext) DeleteRefValues(ref *Ref) {
	if ref == nil {
		return
	}
	switch ref.Type {
	case RefTypeRequestHeader:
		delete(ctx.requestHeaders, strings.ToLower(ref.Name))
		ctx.requestHeadersDirty = true
	case RefTypeRequestQuery:
		delete(ctx.requestQueries, ref.Name)
		ctx.requestQueriesDirty = true
	case RefTypeResponseHeader:
		delete(ctx.responseHeaders, strings.ToLower(ref.Name))
		ctx.responseHeadersDirty = true
	}
}

func (ctx *editorContext) IsRequestHeadersDirty() bool {
	return ctx.requestHeadersDirty
}

func (ctx *editorContext) IsRequestQueriesDirty() bool {
	return ctx.requestQueriesDirty
}

func (ctx *editorContext) IsResponseHeadersDirty() bool {
	return ctx.responseHeadersDirty
}

func (ctx *editorContext) ResetDirtyFlags() {
	ctx.requestHeadersDirty = false
	ctx.requestQueriesDirty = false
	ctx.responseHeadersDirty = false
}

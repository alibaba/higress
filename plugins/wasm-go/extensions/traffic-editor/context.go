package main

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

type EditorContext struct {
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

func (ctx *EditorContext) ResetDirtyFlags() {
	ctx.requestHeadersDirty = false
	ctx.requestQueriesDirty = false
	ctx.responseHeadersDirty = false
}

func (ctx *EditorContext) GetRequestHeader(key string) []string {
	if ctx.requestHeaders == nil {
		return nil
	}
	return ctx.requestHeaders[key]
}

func (ctx *EditorContext) GetRequestQuery(key string) []string {
	if ctx.requestQueries == nil {
		return nil
	}
	return ctx.requestQueries[key]
}

func (ctx *EditorContext) GetResponseHeader(key string) []string {
	if ctx.responseHeaders == nil {
		return nil
	}
	return ctx.responseHeaders[key]
}

func (ctx *EditorContext) GetRefValue(ref *Ref) string {
	values := ctx.GetRefValues(ref)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (ctx *EditorContext) GetRefValues(ref *Ref) []string {
	if ref == nil {
		return nil
	}
	switch ref.Type {
	case refTypeRequestHeader:
		return ctx.GetRequestHeader(ref.Name)
	case refTypeRequestQuery:
		return ctx.GetRequestQuery(ref.Name)
	case refTypeResponseHeader:
		return ctx.GetResponseHeader(ref.Name)
	default:
		return nil
	}
}

func (ctx *EditorContext) SetRefValue(ref *Ref, value string) {
	if ref == nil {
		return
	}
	ctx.SetRefValues(ref, []string{value})
}

func (ctx *EditorContext) SetRefValues(ref *Ref, values []string) {
	if ref == nil {
		return
	}
	switch ref.Type {
	case refTypeRequestHeader:
		ctx.requestHeaders[ref.Name] = values
		ctx.requestHeadersDirty = true
		break
	case refTypeRequestQuery:
		ctx.requestQueries[ref.Name] = values
		ctx.requestQueriesDirty = true
		break
	case refTypeResponseHeader:
		ctx.responseHeaders[ref.Name] = values
		ctx.responseHeadersDirty = true
		break
	}
}

func (ctx *EditorContext) DeleteRefValues(ref *Ref) {
	if ref == nil {
		return
	}
	switch ref.Type {
	case refTypeRequestHeader:
		delete(ctx.requestHeaders, ref.Name)
		ctx.requestHeadersDirty = true
		break
	case refTypeRequestQuery:
		delete(ctx.requestQueries, ref.Name)
		ctx.requestQueriesDirty = true
		break
	case refTypeResponseHeader:
		delete(ctx.responseHeaders, ref.Name)
		ctx.responseHeadersDirty = true
		break
	}
}

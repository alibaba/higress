package engine

import (
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

type SearchResult struct {
	Title   string
	Link    string
	Content string
}

func (result SearchResult) Valid() bool {
	return result.Title != "" && result.Link != "" && result.Content != ""
}

type SearchContext struct {
	EngineType    string
	Querys        []string
	Language      string
	ArxivCategory string
}

type CallArgs struct {
	Method             string
	Url                string
	Headers            [][2]string
	Body               []byte
	TimeoutMillisecond uint32
}

type SearchEngine interface {
	NeedExectue(ctx SearchContext) bool
	Client() wrapper.HttpClient
	CallArgs(ctx SearchContext) CallArgs
	ParseResult(ctx SearchContext, response []byte) []SearchResult
}

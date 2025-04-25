package ocr

import "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"

type CallArgs struct {
	Method             string
	Url                string
	Headers            [][2]string
	Body               []byte
	TimeoutMillisecond uint32
}

type OcrClient interface {
	Client() wrapper.HttpClient
	CallArgs(imageUrl string) CallArgs
	ParseResult(response []byte) string
}

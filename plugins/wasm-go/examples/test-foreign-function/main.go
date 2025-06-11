package main

import (
	pb "github.com/alibaba/higress/plugins/wasm-go/pkg/protos"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/proto"
)

type TestConfig struct {
}

func parseConfig(configJson gjson.Result, config *TestConfig, log wrapper.Log) error {
	return nil
}

func main() {}

func init() {
	wrapper.SetCtx(
		"test-foreign-function",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}
func onHttpResponseHeaders(ctx wrapper.HttpContext, config TestConfig, log wrapper.Log) types.Action {
	proxywasm.RemoveHttpResponseHeader("content-length")
	ctx.DontReadResponseBody()
	d := &pb.InjectEncodedDataToFilterChainArguments{
		Body:      "hello foreign function\n",
		Endstream: true,
	}
	s, _ := proto.Marshal(d)
	_, err := proxywasm.CallForeignFunction("inject_encoded_data_to_filter_chain_on_header", s)
	if err != nil {
		log.Errorf("call inject_encoded_data_to_filter_chain_on_header failed, error: %+v", err)
	}
	return types.ActionPause
}

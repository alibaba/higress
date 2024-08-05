export * from "@higress/proxy-wasm-assemblyscript-sdk/assembly/proxy";
import { SetCtx, HttpContext, ProcessRequestHeadersBy, logger, ParseResult, ParseConfigBy } from "@higress/wasm-assemblyscript/assembly";
import { FilterHeadersStatusValues, send_http_response, stream_context } from "@higress/proxy-wasm-assemblyscript-sdk/assembly"
import { JSON } from "assemblyscript-json/assembly";

class HelloWorldConfig {
}

SetCtx<HelloWorldConfig>("hello-world", [ParseConfigBy<HelloWorldConfig>(parseConfig), ProcessRequestHeadersBy<HelloWorldConfig>(onHttpRequestHeaders)])

function parseConfig(json: JSON.Obj): ParseResult<HelloWorldConfig> {
  return new ParseResult<HelloWorldConfig>(new HelloWorldConfig(), true);
}

function onHttpRequestHeaders(context: HttpContext, config: HelloWorldConfig): FilterHeadersStatusValues {
  stream_context.headers.request.add("hello", "world")
  logger.Debug("set request header")
  send_http_response(200, "hello-world", String.UTF8.encode("[wasm-assemblyscript]hello world"), []);
  return FilterHeadersStatusValues.Continue;
}
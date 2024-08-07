export * from "@higress/proxy-wasm-assemblyscript-sdk/assembly/proxy";
import { SetCtx, HttpContext, ProcessRequestHeadersBy, Logger, ParseResult, ParseConfigBy, RegisteTickFunc } from "@higress/wasm-assemblyscript/assembly";
import { FilterHeadersStatusValues, send_http_response, stream_context } from "@higress/proxy-wasm-assemblyscript-sdk/assembly"
import { JSON } from "assemblyscript-json/assembly";
class HelloWorldConfig {
}

SetCtx<HelloWorldConfig>("hello-world", [ParseConfigBy<HelloWorldConfig>(parseConfig), ProcessRequestHeadersBy<HelloWorldConfig>(onHttpRequestHeaders)])

function parseConfig(json: JSON.Obj): ParseResult<HelloWorldConfig> {
  RegisteTickFunc(2000, () => {
    Logger.Debug("tick 2s");
  })
  RegisteTickFunc(5000, () => {
    Logger.Debug("tick 5s");
  })
  return new ParseResult<HelloWorldConfig>(new HelloWorldConfig(), true);
}

function onHttpRequestHeaders(context: HttpContext, config: HelloWorldConfig): FilterHeadersStatusValues {
  stream_context.headers.request.add("hello", "world");
  Logger.Debug("[hello-world] logger test");
  send_http_response(200, "hello-world", String.UTF8.encode("[wasm-assemblyscript]hello world"), []);
  return FilterHeadersStatusValues.Continue;
}
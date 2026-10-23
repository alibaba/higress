export * from "@higress/proxy-wasm-assemblyscript-sdk/assembly/proxy";
import { SetCtx, HttpContext, ProcessRequestHeadersBy, Logger, ParseConfigBy, ParseResult, ProcessResponseHeadersBy } from "@higress/wasm-assemblyscript/assembly";
import { FilterHeadersStatusValues, Headers, send_http_response, stream_context, HeaderPair } from "@higress/proxy-wasm-assemblyscript-sdk/assembly"
import { JSON } from "assemblyscript-json/assembly";

class CustomResponseConfig {
  statusCode: u32;
  headers: Headers;
  body: ArrayBuffer;
  enableOnStatus: Array<u32>;
  contentType: string;
  constructor() {
    this.statusCode = 200;
    this.headers = [];
    this.body = new ArrayBuffer(0);
    this.enableOnStatus = [];
    this.contentType = "text/plain; charset=utf-8";
  }
}

SetCtx<CustomResponseConfig>(
  "custom-response", 
  [ParseConfigBy<CustomResponseConfig>(parseConfig), 
    ProcessRequestHeadersBy<CustomResponseConfig>(onHttpRequestHeaders),
    ProcessResponseHeadersBy<CustomResponseConfig>(onHttpResponseHeaders),])

function parseConfig(json: JSON.Obj): ParseResult<CustomResponseConfig> {
  let headersArray = json.getArr("headers");
  let config = new CustomResponseConfig();
  if (headersArray != null) {
    for (let i = 0; i < headersArray.valueOf().length; i++) {
      let header = headersArray._arr[i];
      let jsonString = (<JSON.Str>header).toString()
      let kv = jsonString.split("=")
      if (kv.length == 2) {
        let key = kv[0].trim();
        let value = kv[1].trim();
        if (key.toLowerCase() == "content-type") {
          config.contentType = value;
        } else if (key.toLowerCase() == "content-length") {
          continue;
        } else {
          config.headers.push(new HeaderPair(String.UTF8.encode(key), String.UTF8.encode(value)));
        }
      } else {
        Logger.Error("parse header failed");
        return new ParseResult<CustomResponseConfig>(null, false);
      }
    }
  }
  let body = json.getString("body");
  if (body != null) {
    config.body = String.UTF8.encode(body.valueOf());
  }
  config.headers.push(new HeaderPair(String.UTF8.encode("content-type"), String.UTF8.encode(config.contentType)));

  let statusCode = json.getInteger("statusCode");
  if (statusCode != null) {
    config.statusCode = statusCode.valueOf() as u32;
  }

  let enableOnStatus = json.getArr("enableOnStatus");

  if (enableOnStatus != null) {
    for (let i = 0; i < enableOnStatus.valueOf().length; i++) {
      let status = enableOnStatus._arr[i];
      if (status.isInteger) {
        config.enableOnStatus.push((<JSON.Integer>status).valueOf() as u32);
      }
    }
  }
  return new ParseResult<CustomResponseConfig>(config, true);
}

function onHttpRequestHeaders(context: HttpContext, config: CustomResponseConfig): FilterHeadersStatusValues {
  if (config.enableOnStatus.length != 0) {
    return FilterHeadersStatusValues.Continue;
  }
  send_http_response(config.statusCode, "custom-response", config.body, config.headers);
  return FilterHeadersStatusValues.StopIteration;
}

function onHttpResponseHeaders(context: HttpContext, config: CustomResponseConfig): FilterHeadersStatusValues {
  let statusCodeStr = stream_context.headers.response.get(":status")
  if (statusCodeStr == "") {
    Logger.Error("get http response status code failed");
    return FilterHeadersStatusValues.Continue;
  }
  let statusCode = parseInt(statusCodeStr);
  for (let i = 0; i < config.enableOnStatus.length; i++) {
    if (statusCode == config.enableOnStatus[i]) {
      send_http_response(config.statusCode, "custom-response", config.body, config.headers);
    }
  }
  return FilterHeadersStatusValues.Continue;
}

import {
  Cluster
} from "./cluster_wrapper"

import {
  log,
  LogLevelValues,
  Headers,
  HeaderPair,
  root_context,
  BufferTypeValues,
  get_buffer_bytes,
  BaseContext,
  stream_context,
  WasmResultValues,
  RootContext,
  ResponseCallBack
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";

export interface HttpClient {
  get(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  head(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  options(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  post(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  put(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  patch(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  delete(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  connect(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
  trace(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32): boolean;
}

const methodArrayBuffer: ArrayBuffer = String.UTF8.encode(":method");
const pathArrayBuffer: ArrayBuffer = String.UTF8.encode(":path");
const authorityArrayBuffer: ArrayBuffer = String.UTF8.encode(":authority");

const StatusBadGateway: i32 = 502;

export class ClusterClient {
  cluster: Cluster;

  constructor(cluster: Cluster) {
    this.cluster = cluster;
  }

  private httpCall(method: string, path: string, headers: Headers, body: ArrayBuffer, callback: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    if (root_context == null) {
      log(LogLevelValues.error, "Root context is null");
      return false;
    }
    for (let i: i32 = headers.length - 1; i >= 0; i--) {
      const key = String.UTF8.decode(headers[i].key)
      if ((key == ":method") || (key == ":path") || (key == ":authority")) {
        headers.splice(i, 1);
      }
    }

    headers.push(new HeaderPair(methodArrayBuffer, String.UTF8.encode(method)));
    headers.push(new HeaderPair(pathArrayBuffer, String.UTF8.encode(path)));
    headers.push(new HeaderPair(authorityArrayBuffer, String.UTF8.encode(this.cluster.hostName())));

    const result = (root_context as RootContext).httpCall(this.cluster.clusterName(), headers, body, [], timeoutMillisecond, root_context as BaseContext, callback,
      (_origin_context: BaseContext, _numHeaders: u32, body_size: usize, _trailers: u32, callback: ResponseCallBack): void => {
        const respBody = get_buffer_bytes(BufferTypeValues.HttpCallResponseBody, 0, body_size as u32);
        const respHeaders = stream_context.headers.http_callback.get_headers()
        let code = StatusBadGateway;
        let headers = new Array<HeaderPair>();
        for (let i = 0; i < respHeaders.length; i++) {
          const h = respHeaders[i];
          if (String.UTF8.decode(h.key) == ":status") {
            code = <i32>parseInt(String.UTF8.decode(h.value))
          }
          headers.push(new HeaderPair(h.key, h.value));
        }
        log(LogLevelValues.debug, `http call end, code: ${code}, body: ${String.UTF8.decode(respBody)}`)
        callback(code, headers, respBody);
      })
    log(LogLevelValues.debug, `http call start, cluster: ${this.cluster.clusterName()}, method: ${method}, path: ${path}, body: ${String.UTF8.decode(body)}, timeout: ${timeoutMillisecond}`)
    if (result != WasmResultValues.Ok) {
      log(LogLevelValues.error, `http call failed, result: ${result}`)
      return false
    }
    return true
  }

  get(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("GET", path, headers, new ArrayBuffer(0), cb, timeoutMillisecond);
  }

  head(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("HEAD", path, headers, new ArrayBuffer(0), cb, timeoutMillisecond);
  }

  options(path: string, headers: Headers, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("OPTIONS", path, headers, new ArrayBuffer(0), cb, timeoutMillisecond);
  }

  post(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("POST", path, headers, body, cb, timeoutMillisecond);
  }

  put(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("PUT", path, headers, body, cb, timeoutMillisecond);
  }

  patch(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("PATCH", path, headers, body, cb, timeoutMillisecond);
  }

  delete(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("DELETE", path, headers, body, cb, timeoutMillisecond);
  }

  connect(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("CONNECT", path, headers, body, cb, timeoutMillisecond);
  }

  trace(path: string, headers: Headers, body: ArrayBuffer, cb: ResponseCallBack, timeoutMillisecond: u32 = 500): boolean {
    return this.httpCall("TRACE", path, headers, body, cb, timeoutMillisecond);
  }
}

import {
  stream_context,
  log,
  LogLevelValues
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";

export function getRequestScheme(): string {
  let scheme: string  = stream_context.headers.request.get(":scheme");
  if (scheme == "") {
    log(LogLevelValues.error, "Parse request scheme failed");
  }
  return scheme;
}

export function getRequestHost(): string {
  let host: string = stream_context.headers.request.get(":authority");
  if (host == "") {
    log(LogLevelValues.error, "Parse request host failed");
  }
  return host;
}

export function getRequestPath(): string {
  let path: string = stream_context.headers.request.get(":path");
  if (path == "") {
    log(LogLevelValues.error, "Parse request path failed");
  }
  return path;
}

export function getRequestMethod(): string {
  let method: string = stream_context.headers.request.get(":method");
  if (method == "") {
    log(LogLevelValues.error, "Parse request method failed");
  }
  return method;
}

export function isBinaryRequestBody(): boolean {
  let contentType: string = stream_context.headers.request.get("content-type");
  if (contentType != "" && (contentType.includes("octet-stream") || contentType.includes("grpc"))) {
    return true;
  }

  let encoding: string = stream_context.headers.request.get("content-encoding");
  if (encoding != "") {
    return true;
  }

  return false;
}

export function isBinaryResponseBody(): boolean {
  let contentType: string = stream_context.headers.response.get("content-type");
  if (contentType != "" && (contentType.includes("octet-stream") || contentType.includes("grpc"))) {
    return true;
  }

  let encoding: string = stream_context.headers.response.get("content-encoding");
  if (encoding != "") {
    return true;
  }

  return false;
}
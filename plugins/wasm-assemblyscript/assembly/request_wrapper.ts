import {
  stream_context,
  log,
  LogLevelValues
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";

export function getRequestScheme(): string {
  let scheme: string | null = stream_context.headers.request.get(":scheme");
  if (scheme == null) {
    log(LogLevelValues.error, "Parse request scheme failed");
    return "";
  }
  return scheme;
}

export function getRequestHost(): string {
  let host: string | null = stream_context.headers.request.get(":authority");
  if (host == null) {
    log(LogLevelValues.error, "Parse request host failed");
    return "";
  }
  return host;
}

export function getRequestPath(): string {
  let path: string | null = stream_context.headers.request.get(":path");
  if (path == null) {
    log(LogLevelValues.error, "Parse request path failed");
    return "";
  }
  return path;
}

export function getRequestMethod(): string {
  let method: string | null = stream_context.headers.request.get(":method");
  if (method == null) {
    log(LogLevelValues.error, "Parse request method failed");
    return "";
  }
  return method;
}

export function isBinaryRequestBody(): boolean {
  let contentType: string | null = stream_context.headers.request.get("content-type");
  if (contentType != null && (contentType.includes("octet-stream") || contentType.includes("grpc"))) {
    return true;
  }

  let encoding: string | null = stream_context.headers.request.get("content-encoding");
  if (encoding != null && encoding != "") {
    return true;
  }

  return false;
}

export function isBinaryResponseBody(): boolean {
  let contentType: string | null = stream_context.headers.response.get("content-type");
  if (contentType != null && (contentType.includes("octet-stream") || contentType.includes("grpc"))) {
    return true;
  }

  let encoding: string | null = stream_context.headers.response.get("content-encoding");
  if (encoding != null && encoding != "") {
    return true;
  }

  return false;
}
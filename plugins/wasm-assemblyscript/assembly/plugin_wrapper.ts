import { Log } from "./log_wrapper";
import {
  Context,
  FilterHeadersStatusValues,
  RootContext,
  setRootContext,
  proxy_set_effective_context,
  log,
  LogLevelValues,
  FilterDataStatusValues,
  get_buffer_bytes,
  BufferTypeValues,
  set_tick_period_milliseconds,
  get_current_time_nanoseconds
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";
import {
  getRequestHost,
  getRequestMethod,
  getRequestPath,
  getRequestScheme,
  isBinaryRequestBody,
} from "./request_wrapper";
import { RuleMatcher, ParseResult } from "./rule_matcher";
import { JSON } from "assemblyscript-json/assembly";

export function SetCtx<PluginConfig>(
  pluginName: string,
  setFuncs: usize[] = []
): void {
  const rootContextId = 1
  setRootContext(new CommonRootCtx<PluginConfig>(rootContextId, pluginName, setFuncs));
}

export interface HttpContext {
  Scheme(): string;
  Host(): string;
  Path(): string;
  Method(): string;
  SetContext(key: string, value: usize): void;
  GetContext(key: string): usize;
  DontReadRequestBody(): void;
  DontReadResponseBody(): void;
}

type ParseConfigFunc<PluginConfig> = (
  json: JSON.Obj,
) => ParseResult<PluginConfig>;
type OnHttpHeadersFunc<PluginConfig> = (
  context: HttpContext,
  config: PluginConfig,
) => FilterHeadersStatusValues;
type OnHttpBodyFunc<PluginConfig> = (
  context: HttpContext,
  config: PluginConfig,
  body: ArrayBuffer,
) => FilterDataStatusValues;


export var Logger: Log = new Log("");

class CommonRootCtx<PluginConfig> extends RootContext {
  pluginName: string;
  hasCustomConfig: boolean;
  ruleMatcher: RuleMatcher<PluginConfig>;
  parseConfig: ParseConfigFunc<PluginConfig> | null;
  onHttpRequestHeaders: OnHttpHeadersFunc<PluginConfig> | null;
  onHttpRequestBody: OnHttpBodyFunc<PluginConfig> | null;
  onHttpResponseHeaders: OnHttpHeadersFunc<PluginConfig> | null;
  onHttpResponseBody: OnHttpBodyFunc<PluginConfig> | null;
  onTickFuncs: Array<TickFuncEntry>;

  constructor(context_id: u32, pluginName: string, setFuncs: usize[]) {
    super(context_id);
    this.pluginName = pluginName;
    Logger = new Log(pluginName);
    this.hasCustomConfig = true;
    this.onHttpRequestHeaders = null;
    this.onHttpRequestBody = null;
    this.onHttpResponseHeaders = null;
    this.onHttpResponseBody = null;
    this.parseConfig = null;
    this.ruleMatcher = new RuleMatcher<PluginConfig>();
    this.onTickFuncs = new Array<TickFuncEntry>();
    for (let i = 0; i < setFuncs.length; i++) {
      changetype<Closure<PluginConfig>>(setFuncs[i]).lambdaFn(
        setFuncs[i],
        this
      );
    }
    if (this.parseConfig == null) {
      this.hasCustomConfig = false;
      this.parseConfig = (json: JSON.Obj): ParseResult<PluginConfig> =>{ return new ParseResult<PluginConfig>(null, true); };
    }
  }

  createContext(context_id: u32): Context {
    return new CommonCtx<PluginConfig>(context_id, this);
  }

  onConfigure(configuration_size: u32): boolean {
    super.onConfigure(configuration_size);
    const data = this.getConfiguration();
    let jsonData: JSON.Obj = new JSON.Obj();
    if (data == "{}") {
      if (this.hasCustomConfig) {
        log(LogLevelValues.warn, "config is empty, but has ParseConfigFunc");
      } 
    } else {
      const parseData = JSON.parse(data);
      if (parseData.isObj) {
        jsonData = changetype<JSON.Obj>(JSON.parse(data));
      } else {
        log(LogLevelValues.error, "parse json data failed")
        return false;
      }
    }

    if (!this.ruleMatcher.parseRuleConfig(jsonData, this.parseConfig as ParseConfigFunc<PluginConfig>)) {
      return false;
    }

    if (globalOnTickFuncs.length > 0) {
      this.onTickFuncs = globalOnTickFuncs;
      set_tick_period_milliseconds(100);
    }
    return true;
  }

  onTick(): void {
    for (let i = 0; i < this.onTickFuncs.length; i++) {
      const tickFuncEntry = this.onTickFuncs[i];
      const now = getCurrentTimeMilliseconds();
      if (tickFuncEntry.lastExecuted + tickFuncEntry.tickPeriod <= now) {
        tickFuncEntry.tickFunc();
        tickFuncEntry.lastExecuted = getCurrentTimeMilliseconds();
      }
    }
  }
}

function getCurrentTimeMilliseconds(): u64 {
  return get_current_time_nanoseconds() / 1000000;
}

class TickFuncEntry {
  lastExecuted: u64;
  tickPeriod: u64;
  tickFunc: () => void;

  constructor(lastExecuted: u64, tickPeriod: u64, tickFunc: () => void) {
    this.lastExecuted = lastExecuted;
    this.tickPeriod = tickPeriod;
    this.tickFunc = tickFunc;
  }
}

var globalOnTickFuncs = new Array<TickFuncEntry>();

export function RegisteTickFunc(tickPeriod: i64, tickFunc: () => void): void {
  globalOnTickFuncs.push(new TickFuncEntry(0, tickPeriod, tickFunc));
}

class Closure<PluginConfig> {
  lambdaFn: (closure: usize, ctx: CommonRootCtx<PluginConfig>) => void;
  parseConfigFunc: ParseConfigFunc<PluginConfig> | null;
  onHttpHeadersFunc: OnHttpHeadersFunc<PluginConfig> | null;
  OnHttpBodyFunc: OnHttpBodyFunc<PluginConfig> | null;

  constructor(
    lambdaFn: (closure: usize, ctx: CommonRootCtx<PluginConfig>) => void
  ) {
    this.lambdaFn = lambdaFn;
    this.parseConfigFunc = null;
    this.onHttpHeadersFunc = null;
    this.OnHttpBodyFunc = null;
  }

  setParseConfigFunc(f: ParseConfigFunc<PluginConfig>): void {
    this.parseConfigFunc = f;
  }

  setHttpHeadersFunc(f: OnHttpHeadersFunc<PluginConfig>): void {
    this.onHttpHeadersFunc = f;
  }

  setHttpBodyFunc(f: OnHttpBodyFunc<PluginConfig>): void {
    this.OnHttpBodyFunc = f;
  }
}

export function ParseConfigBy<PluginConfig>(
  f: ParseConfigFunc<PluginConfig>
): usize {
  const lambdaFn = function (
    closure: usize,
    ctx: CommonRootCtx<PluginConfig>
  ): void {
    const f = changetype<Closure<PluginConfig>>(closure).parseConfigFunc;
    if (f != null) {
      ctx.parseConfig = f;
    }
  };
  const closure = new Closure<PluginConfig>(lambdaFn);
  closure.setParseConfigFunc(f);
  return changetype<usize>(closure);
}

export function ProcessRequestHeadersBy<PluginConfig>(
  f: OnHttpHeadersFunc<PluginConfig>
): usize {
  const lambdaFn = function (
    closure: usize,
    ctx: CommonRootCtx<PluginConfig>
  ): void {
    const f = changetype<Closure<PluginConfig>>(closure).onHttpHeadersFunc;
    if (f != null) {
      ctx.onHttpRequestHeaders = f;
    }
  };
  const closure = new Closure<PluginConfig>(lambdaFn);
  closure.setHttpHeadersFunc(f);
  return changetype<usize>(closure);
}

export function ProcessRequestBodyBy<PluginConfig>(
  f: OnHttpBodyFunc<PluginConfig>
): usize {
  const lambdaFn = function (
    closure: usize,
    ctx: CommonRootCtx<PluginConfig>
  ): void {
    const f = changetype<Closure<PluginConfig>>(closure).OnHttpBodyFunc;
    if (f != null) {
      ctx.onHttpRequestBody = f;
    }
  };
  const closure = new Closure<PluginConfig>(lambdaFn);
  closure.setHttpBodyFunc(f);
  return changetype<usize>(closure);
}

export function ProcessResponseHeadersBy<PluginConfig>(
  f: OnHttpHeadersFunc<PluginConfig>
): usize {
  const lambdaFn = function (
    closure: usize,
    ctx: CommonRootCtx<PluginConfig>
  ): void {
    const f = changetype<Closure<PluginConfig>>(closure).onHttpHeadersFunc;
    if (f != null) {
      ctx.onHttpResponseHeaders = f;
    }
  };
  const closure = new Closure<PluginConfig>(lambdaFn);
  closure.setHttpHeadersFunc(f);
  return changetype<usize>(closure);
}

export function ProcessResponseBodyBy<PluginConfig>(
  f: OnHttpBodyFunc<PluginConfig>
): usize {
  const lambdaFn = function (
    closure: usize,
    ctx: CommonRootCtx<PluginConfig>
  ): void {
    const f = changetype<Closure<PluginConfig>>(closure).OnHttpBodyFunc;
    if (f != null) {
      ctx.onHttpResponseBody = f;
    }
  };
  const closure = new Closure<PluginConfig>(lambdaFn);
  closure.setHttpBodyFunc(f);
  return changetype<usize>(closure);
}

class CommonCtx<PluginConfig> extends Context implements HttpContext {
  commonRootCtx: CommonRootCtx<PluginConfig>;
  config: PluginConfig |null;
  needRequestBody: boolean;
  needResponseBody: boolean;
  requestBodySize: u32;
  responseBodySize: u32;
  contextID: u32;
  userContext: Map<string, usize>;

  constructor(context_id: u32, root_context: CommonRootCtx<PluginConfig>) {
    super(context_id, root_context);
    this.userContext = new Map<string, usize>();
    this.commonRootCtx = root_context;
    this.contextID = context_id;
    this.requestBodySize = 0;
    this.responseBodySize = 0;
    this.config = null
    if (this.commonRootCtx.onHttpRequestHeaders != null) {
      this.needResponseBody = true;
    } else {
      this.needResponseBody = false;
    }
    if (this.commonRootCtx.onHttpRequestBody != null) {
      this.needRequestBody = true;
    } else {
      this.needRequestBody = false;
    }
  }

  SetContext(key: string, value: usize): void {
    this.userContext.set(key, value);
  }

  GetContext(key: string): usize {
    return this.userContext.get(key);
  }

  Scheme(): string {
    proxy_set_effective_context(this.contextID);
    return getRequestScheme();
  }

  Host(): string {
    proxy_set_effective_context(this.contextID);
    return getRequestHost();
  }

  Path(): string {
    proxy_set_effective_context(this.contextID);
    return getRequestPath();
  }

  Method(): string {
    proxy_set_effective_context(this.contextID);
    return getRequestMethod();
  }

  DontReadRequestBody(): void {
    this.needRequestBody = false;
  }

  DontReadResponseBody(): void {
    this.needResponseBody = false;
  }

  onRequestHeaders(_a: u32, _end_of_stream: boolean): FilterHeadersStatusValues {
    const parseResult = this.commonRootCtx.ruleMatcher.getMatchConfig();
    if (parseResult.success == false) {
      log(LogLevelValues.error, "get match config failed");
      return FilterHeadersStatusValues.Continue;
    }
    this.config = parseResult.pluginConfig;

    if (isBinaryRequestBody()) {
      this.needRequestBody = false;
    }

    if (this.commonRootCtx.onHttpRequestHeaders == null) {
      return FilterHeadersStatusValues.Continue;
    }
    return this.commonRootCtx.onHttpRequestHeaders(
      this,
      this.config as PluginConfig
    );
  }

  onRequestBody(
    body_buffer_length: usize,
    end_of_stream: boolean
  ): FilterDataStatusValues {
    if (this.config == null || !this.needRequestBody) {
      return FilterDataStatusValues.Continue;
    }

    if (this.commonRootCtx.onHttpRequestBody == null) {
      return FilterDataStatusValues.Continue;
    }
    this.requestBodySize += body_buffer_length as u32;

    if (!end_of_stream) {
      return FilterDataStatusValues.StopIterationAndBuffer;
    }

    const body = get_buffer_bytes(
      BufferTypeValues.HttpRequestBody,
      0,
      this.requestBodySize
    );

    return this.commonRootCtx.onHttpRequestBody(
      this,
      this.config as PluginConfig,
      body
    );
  }

  onResponseHeaders(_a: u32, _end_of_stream: bool): FilterHeadersStatusValues {
    if (this.config == null) {
      return FilterHeadersStatusValues.Continue;
    }

    if (isBinaryRequestBody()) {
      this.needResponseBody = false;
    }

    if (this.commonRootCtx.onHttpResponseHeaders == null) {
      return FilterHeadersStatusValues.Continue;
    }

    return this.commonRootCtx.onHttpResponseHeaders(
      this,
      this.config as PluginConfig
    );
  }

  onResponseBody(
    body_buffer_length: usize,
    end_of_stream: bool
  ): FilterDataStatusValues {
    if (this.config == null) {
      return FilterDataStatusValues.Continue;
    }

    if (this.commonRootCtx.onHttpResponseBody == null) {
      return FilterDataStatusValues.Continue;
    }

    if (!this.needResponseBody) {
      return FilterDataStatusValues.Continue;
    }

    this.responseBodySize += body_buffer_length as u32;

    if (!end_of_stream) {
      return FilterDataStatusValues.StopIterationAndBuffer;
    }
    const body = get_buffer_bytes(
      BufferTypeValues.HttpResponseBody,
      0,
      this.responseBodySize
    );

    return this.commonRootCtx.onHttpResponseBody(
      this,
      this.config as PluginConfig,
      body
    );
  }
}

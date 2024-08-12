import { getRequestHost } from "./request_wrapper";
import {
  get_property,
  LogLevelValues,
  log,
  WasmResultValues,
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";
import { JSON } from "assemblyscript-json/assembly";

enum Category {
  Route,
  Host,
  RoutePrefix,
  Service
}

enum MatchType {
  Prefix,
  Exact,
  Suffix,
}

const RULES_KEY: string = "_rules_";
const MATCH_ROUTE_KEY: string = "_match_route_";
const MATCH_DOMAIN_KEY: string = "_match_domain_";
const MATCH_SERVICE_KEY: string = "_match_service_";
const MATCH_ROUTE_PREFIX_KEY: string = "_match_route_prefix_"

class HostMatcher {
  matchType: MatchType;
  host: string;

  constructor(matchType: MatchType, host: string) {
    this.matchType = matchType;
    this.host = host;
  }
}

class RuleConfig<PluginConfig> {
  category:      Category;
  routes!:       Map<string, boolean>;
  services!:     Map<string, boolean>;
  routePrefixs!: Map<string, boolean>;
  hosts!:        Array<HostMatcher>;
  config:        PluginConfig | null;

  constructor() {
    this.category = Category.Route;
    this.config = null;
  }
}

export class ParseResult<PluginConfig> {
  pluginConfig: PluginConfig | null;
  success: boolean;
  constructor(pluginConfig: PluginConfig | null, success: boolean) {
    this.pluginConfig = pluginConfig;
    this.success = success;
  }
}

export class RuleMatcher<PluginConfig> {
  ruleConfig: Array<RuleConfig<PluginConfig>>;
  globalConfig: PluginConfig | null;
  hasGlobalConfig: boolean;

  constructor() {
    this.ruleConfig = new Array<RuleConfig<PluginConfig>>();
    this.globalConfig = null;
    this.hasGlobalConfig = false;
  }

  getMatchConfig(): ParseResult<PluginConfig> {
    const host = getRequestHost();
    if (host == "") {
      return new ParseResult<PluginConfig>(null, false);
    }
    let result = get_property("route_name");
    if (result.status != WasmResultValues.Ok && result.status != WasmResultValues.NotFound) {
      return new ParseResult<PluginConfig>(null, false);
    }
    const routeName = String.UTF8.decode(result.returnValue);

    result = get_property("cluster_name");
    if (result.status != WasmResultValues.Ok && result.status != WasmResultValues.NotFound) {
      return new ParseResult<PluginConfig>(null, false);
    }
    const serviceName = String.UTF8.decode(result.returnValue);

    for (let i = 0; i < this.ruleConfig.length; i++) {
      const rule = this.ruleConfig[i];
      // category == Host
      if (rule.category == Category.Host) {
        if (this.hostMatch(rule, host)) {
          log(LogLevelValues.debug, "getMatchConfig: match host " + host);
          return new ParseResult<PluginConfig>(rule.config, true);
        }
      }
      // category == Route
      if (rule.category == Category.Route) {
        if (rule.routes.has(routeName)) {
          log(LogLevelValues.debug, "getMatchConfig: match route " + routeName);
          return new ParseResult<PluginConfig>(rule.config, true);
        }
      }
      // category == RoutePrefix
      if (rule.category == Category.RoutePrefix) {
        for (let i = 0; i < rule.routePrefixs.keys().length; i++) {
          const routePrefix = rule.routePrefixs.keys()[i];
          if (routeName.startsWith(routePrefix)) {
            return new ParseResult<PluginConfig>(rule.config, true);
          }
        }
      }
      // category == Cluster
      if (this.serviceMatch(rule, serviceName)) {
        return new ParseResult<PluginConfig>(rule.config, true);
      }
    }

    if (this.hasGlobalConfig) {
      return new ParseResult<PluginConfig>(this.globalConfig, true);
    }
    return new ParseResult<PluginConfig>(null, false);
  }

  parseRuleConfig(
    config: JSON.Obj,
    parsePluginConfig: (json: JSON.Obj) => ParseResult<PluginConfig>
  ): boolean {
    const obj = config;
    let keyCount = obj.keys.length;
    if (keyCount == 0) {
      this.hasGlobalConfig = true;
      const parseResult = parsePluginConfig(config);
      if (parseResult.success) {
        this.globalConfig = parseResult.pluginConfig;
        return true;
      } else {
        return false;
      }
    }

    let rules: JSON.Arr | null = null;
    if (obj.has(RULES_KEY)) {
      rules = obj.getArr(RULES_KEY);
      keyCount--;
    }

    if (keyCount > 0) {
      const parseResult = parsePluginConfig(config);
      if (parseResult.success) {
        this.globalConfig = parseResult.pluginConfig;
        this.hasGlobalConfig = true;
      }
    }

    if (!rules) {
      if (this.hasGlobalConfig) {
        return true;
      }
      log(LogLevelValues.error, "parse config failed, no valid rules; global config parse error");
      return false;
    }

    const rulesArray = rules.valueOf();
    for (let i = 0; i < rulesArray.length; i++) {
      if (!rulesArray[i].isObj) {
        log(LogLevelValues.error, "parse rule failed, rules must be an array of objects");
        continue;
      }
      const ruleJson = changetype<JSON.Obj>(rulesArray[i]);
      const rule = new RuleConfig<PluginConfig>();
      const parseResult = parsePluginConfig(ruleJson);
      if (parseResult.success) {
        rule.config = parseResult.pluginConfig;
      } else {
        return false;
      }

      rule.routes = this.parseRouteMatchConfig(ruleJson);
      rule.hosts = this.parseHostMatchConfig(ruleJson);
      rule.services = this.parseServiceMatchConfig(ruleJson);
      rule.routePrefixs = this.parseRoutePrefixMatchConfig(ruleJson);

      const noRoute = rule.routes.size == 0;
      const noHosts = rule.hosts.length == 0;
      const noServices = rule.services.size == 0;
      const noRoutePrefixs = rule.routePrefixs.size == 0;

      if ((boolToInt(noRoute) + boolToInt(noHosts) + boolToInt(noServices) + boolToInt(noRoutePrefixs)) != 3) {
        log(LogLevelValues.error, "there is only one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration.");
        return false;
      }
      if (!noRoute) {
        rule.category = Category.Route;
      } else if (!noHosts) {
        rule.category = Category.Host;
      } else if (!noServices) {
        rule.category = Category.Service;
      } else {
        rule.category = Category.RoutePrefix;
      }
      this.ruleConfig.push(rule);
    }
    return true;
  }

  parseRouteMatchConfig(config: JSON.Obj): Map<string, boolean> {
    const keys = config.getArr(MATCH_ROUTE_KEY);
    const routes = new Map<string, boolean>();
    if (keys) {
      const array = keys.valueOf();
      for (let i = 0; i < array.length; i++) {
        const key = array[i].toString();
        if (key != "") {
          routes.set(key, true);
        }
      }
    }
    return routes;
  }

  parseRoutePrefixMatchConfig(config: JSON.Obj): Map<string, boolean> {
    const keys = config.getArr(MATCH_ROUTE_PREFIX_KEY);
    const routePrefixs = new Map<string, boolean>();
    if (keys) {
      const array = keys.valueOf();
      for (let i = 0; i < array.length; i++) {
        const key = array[i].toString();
        if (key != "") {
          routePrefixs.set(key, true);
        }
      }
    }
    return routePrefixs;
  }

  parseServiceMatchConfig(config: JSON.Obj): Map<string, boolean> {
    const keys = config.getArr(MATCH_SERVICE_KEY);
    const clusters = new Map<string, boolean>();
    if (keys) {
      const array = keys.valueOf();
      for (let i = 0; i < array.length; i++) {
        const key = array[i].toString();
        if (key != "") {
          clusters.set(key, true);
        }
      }
    }
    return clusters;
  }

  parseHostMatchConfig(config: JSON.Obj): Array<HostMatcher> {
    const hostMatchers = new Array<HostMatcher>();
    const keys = config.getArr(MATCH_DOMAIN_KEY);
    if (keys !== null) {
      const array = keys.valueOf();
      for (let i = 0; i < array.length; i++) {
        const item = array[i].toString(); // Assuming the array has string elements
        let hostMatcher: HostMatcher;
        if (item.startsWith("*")) {
          hostMatcher = new HostMatcher(MatchType.Suffix, item.substr(1));
        } else if (item.endsWith("*")) {
          hostMatcher = new HostMatcher(
            MatchType.Prefix,
            item.substr(0, item.length - 1)
          );
        } else {
          hostMatcher = new HostMatcher(MatchType.Exact, item);
        }
        hostMatchers.push(hostMatcher);
      }
    }
    return hostMatchers;
  }

  stripPortFromHost(reqHost: string): string {
    // Port removing code is inspired by
    // https://github.com/envoyproxy/envoy/blob/v1.17.0/source/common/http/header_utility.cc#L219
    let portStart: i32 = reqHost.lastIndexOf(":");
    if (portStart != -1) {
      // According to RFC3986 v6 address is always enclosed in "[]".
      // section 3.2.2.
      let v6EndIndex: i32 = reqHost.lastIndexOf("]");
      if (v6EndIndex == -1 || v6EndIndex < portStart) {
        if (portStart + 1 <= reqHost.length) {
          return reqHost.substring(0, portStart);
        }
      }
    }
    return reqHost;
  }

  hostMatch(rule: RuleConfig<PluginConfig>, reqHost: string): boolean {
    reqHost = this.stripPortFromHost(reqHost);
    for (let i = 0; i < rule.hosts.length; i++) {
      let hostMatch = rule.hosts[i];
      switch (hostMatch.matchType) {
        case MatchType.Suffix:
          if (reqHost.endsWith(hostMatch.host)) {
            return true;
          }
          break;
        case MatchType.Prefix:
          if (reqHost.startsWith(hostMatch.host)) {
            return true;
          }
          break;
        case MatchType.Exact:
          if (reqHost == hostMatch.host) {
            return true;
          }
          break;
        default:
          return false;
      }
    }
    return false;
  }

  serviceMatch(rule: RuleConfig<PluginConfig>, serviceName: string): boolean {
    const parts = serviceName.split('|');
    if (parts.length != 4) {
      return false;
    }
    const port = parts[1];
    const fqdn = parts[3];
    for (let i = 0; i < rule.services.keys().length; i++) {
      let configServiceName = rule.services.keys()[i];
      let colonIndex = configServiceName.lastIndexOf(':');
      if (colonIndex != -1) {
          let configFQDN = configServiceName.slice(0, colonIndex);
          let configPort = configServiceName.slice(colonIndex + 1);
          if (fqdn == configFQDN && port == configPort) return true;
      } else if (fqdn == configServiceName) {
          return true;
      }
    }
    return false;
  }
}

function boolToInt(value: boolean): i32 {
  return value ? 1 : 0;
}
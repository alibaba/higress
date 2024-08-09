import {
  log,
  LogLevelValues,
  get_property,
  WasmResultValues,
} from "@higress/proxy-wasm-assemblyscript-sdk/assembly";
import { getRequestHost } from "./request_wrapper";
  
export abstract class Cluster {
  abstract clusterName(): string;
  abstract hostName(): string;
}
  
export class RouteCluster extends Cluster {
  host: string;
  constructor(host: string = "") {
    super();
    this.host = host;
  }
  
  clusterName(): string {
    let result = get_property("cluster_name");
    if (result.status != WasmResultValues.Ok) {
      log(LogLevelValues.error, "get route cluster failed");
      return "";
    }
    return String.UTF8.decode(result.returnValue);
  }
  
  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return getRequestHost();
  }
}
  
export class K8sCluster extends Cluster {
  serviceName: string;
  namespace: string;
  port: i64;
  version: string;
  host: string;

  constructor(
    serviceName: string,
    namespace: string,
    port: i64,
    version: string = "",
    host: string = ""
  ) {
    super();
    this.serviceName = serviceName;
    this.namespace = namespace;
    this.port = port;
    this.version = version;
    this.host = host;
  }

  clusterName(): string {
    let namespace = this.namespace != "" ? this.namespace : "default";
    return `outbound|${this.port}|${this.version}|${this.serviceName}.${namespace}.svc.cluster.local`;
  }

  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return `${this.serviceName}.${this.namespace}.svc.cluster.local`;
  }
}

export class NacosCluster extends Cluster {
  serviceName: string;
  group: string;
  namespaceID: string;
  port: i64;
  isExtRegistry: boolean;
  version: string;
  host: string;

  constructor(
    serviceName: string,
    namespaceID: string,
    port: i64,
    // use DEFAULT-GROUP by default
    group: string = "DEFAULT-GROUP",
    // set true if use edas/sae registry
    isExtRegistry: boolean = false,
    version: string = "",
    host: string = ""
  ) {
    super();
    this.serviceName = serviceName;
    this.group = group.replace("_", "-");
    this.namespaceID = namespaceID;
    this.port = port;
    this.isExtRegistry = isExtRegistry;
    this.version = version;
    this.host = host;
  }

  clusterName(): string {
    let tail = "nacos" + (this.isExtRegistry ? "-ext" : "");
    return `outbound|${this.port}|${this.version}|${this.serviceName}.${this.group}.${this.namespaceID}.${tail}`;
  }

  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return this.serviceName;
  }
}

export class StaticIpCluster extends Cluster {
  serviceName: string;
  port: i64;
  host: string;

  constructor(serviceName: string, port: i64, host: string = "") {
    super()
    this.serviceName = serviceName;
    this.port = port;
    this.host = host;
  }

  clusterName(): string {
    return `outbound|${this.port}||${this.serviceName}.static`;
  }

  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return this.serviceName;
  }
}

export class DnsCluster extends Cluster {
  serviceName: string;
  domain: string;
  port: i64;

  constructor(serviceName: string, domain: string, port: i64) {
    super();
    this.serviceName = serviceName;
    this.domain = domain;
    this.port = port;
  }

  clusterName(): string {
    return `outbound|${this.port}||${this.serviceName}.dns`;
  }

  hostName(): string {
    return this.domain;
  }
}

export class ConsulCluster extends Cluster {
  serviceName: string;
  datacenter: string;
  port: i64;
  host: string;

  constructor(
    serviceName: string,
    datacenter: string,
    port: i64,
    host: string = ""
  ) {
    super();
    this.serviceName = serviceName;
    this.datacenter = datacenter;
    this.port = port;
    this.host = host;
  }

  clusterName(): string {
    return `outbound|${this.port}||${this.serviceName}.${this.datacenter}.consul`;
  }

  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return this.serviceName;
  }
}

export class FQDNCluster extends Cluster {
  fqdn: string;
  host: string;
  port: i64;

  constructor(fqdn: string, port: i64, host: string = "") {
    super();
    this.fqdn = fqdn;
    this.host = host;
    this.port = port;
  }

  clusterName(): string {
    return `outbound|${this.port}||${this.fqdn}`;
  }

  hostName(): string {
    if (this.host != "") {
      return this.host;
    }
    return this.fqdn;
  }
}

#!/usr/bin/env python3
"""Generate deterministic static Envoy configurations for the runtime harness."""

import json
import os
from copy import deepcopy
from pathlib import Path


OUT = Path(os.environ.get("RUNTIME_OUT", "/evidence"))


TOOLS = [{
    "name": "proxy_echo",
    "description": "Deterministic observable proxy tool",
    "args": [{"name": "value", "description": "Value to echo", "type": "string", "required": True}],
}]


SCHEMA_COMPATIBILITY = {
    "server": {"name": "schema-compatibility", "type": "rest"},
    "tools": [
        {
            "name": "getTransactionRecordListV2",
            "description": "Pinned schema compatibility fixture",
            "args": [
                {"name": "transactionId", "description": "Transaction identifier", "type": "string", "required": True, "position": "path"},
                {"name": "page", "description": "Page number", "type": "integer", "default": 10, "position": "query"},
                {"name": "X-Compat-Flag", "description": "Compatibility flag", "type": "boolean", "default": False, "position": "header"},
                {"name": "businessType", "description": "Business types", "type": "array", "enum": ["SALE", "REFUND"], "items": {"type": "string"}, "position": "body"},
                {"name": "payload", "description": "Deterministic payload", "type": "object", "properties": {"amount": {"type": "integer"}}, "position": "body"},
            ],
            "requestTemplate": {
                "url": "http://backend-primary:8080/compat/{transactionId}?fixed=yes",
                "method": "POST",
                "argsToJsonBody": True,
            },
        },
        {
            "name": "compat_health",
            "description": "Unrelated valid compatibility tool",
            "requestTemplate": {"url": "http://backend-primary:8080/compat/health", "method": "GET"},
        },
    ],
}

VALID_SCHEMA_COMPATIBILITY = deepcopy(SCHEMA_COMPATIBILITY)
del VALID_SCHEMA_COMPATIBILITY["tools"][0]["args"][3]["enum"]

MALFORMED_NON_SCHEMA = {
    "server": {"name": "malformed-non-schema", "type": "rest"},
    "tools": [{
        "name": "malformed_template",
        "description": "Independent historical configuration error control",
        "args": [{"name": "value", "type": "string"}],
        "requestTemplate": {"url": "{{", "method": "GET"},
    }],
}


def proxy(name, strategy=None, target="backend-primary", auth=False):
    server = {
        "name": name,
        "type": "mcp-proxy",
        "transport": "http",
        "mcpServerURL": f"http://{target}:8080/{'modern' if strategy == 'modern' else 'legacy'}",
    }
    if strategy is not None:
        server["protocolStrategy"] = strategy
    if auth:
        server["securitySchemes"] = [{
            "id": "RuntimeBearer", "type": "http", "scheme": "bearer",
            "defaultCredential": "runtime-upstream-token",
        }]
        server["defaultUpstreamSecurity"] = {"id": "RuntimeBearer"}
    return {"server": server, "tools": TOOLS}


LISTENERS = [
    (10000, "registered-amap", "backend-primary", {"server": {"name": "amap-tools", "config": {"apiKey": "runtime-key"}}}),
    (10001, "runtime-rest", "backend-primary", {
        "server": {"name": "runtime-rest", "type": "rest"},
        "tools": [{
            "name": "get_weather", "description": "Query deterministic weather",
            "args": [{"name": "city", "description": "City", "type": "string", "required": True}],
            "requestTemplate": {"url": "http://backend-primary:8080/rest/weather", "method": "GET", "argsToUrlParam": True},
        }],
    }),
    (10002, "composed-tools", "backend-primary", {
        "server": {"name": "amap-tools", "config": {"apiKey": "runtime-key"}},
        "_rules_": [{
            "_match_domain_": ["mcp.runtime.test"],
            "toolSet": {"name": "runtime-composed", "serverTools": [{"serverName": "amap-tools", "tools": ["maps_weather"]}]},
        }],
    }),
    (10003, "proxy-modern", "backend-primary", proxy("proxy-modern", "modern")),
    (10004, "proxy-legacy", "backend-primary", proxy("proxy-legacy", "legacy")),
    (10005, "proxy-default-legacy", "backend-primary", proxy("proxy-default-legacy")),
    (10006, "proxy-auth-modern", "backend-primary", proxy("proxy-auth-modern", "modern", auth=True)),
    (10007, "proxy-secondary-modern", "backend-secondary", proxy("proxy-secondary-modern", "modern", target="backend-secondary")),
    (10008, "schema-compatibility", "backend-primary", SCHEMA_COMPATIBILITY),
]


def block(text, spaces):
    prefix = " " * spaces
    return "\n".join(prefix + line for line in text.splitlines())


def listener_yaml(port, name, cluster, config, wasm_file="plugin.wasm"):
    config_json = json.dumps(config, sort_keys=True, separators=(",", ":"))
    return f'''  - name: {name}
    address:
      socket_address: {{address: 0.0.0.0, port_value: {port}}}
    filter_chains:
      - filters:
          - name: envoy.filters.network.http_connection_manager
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
              stat_prefix: {name}
              access_log:
                - name: envoy.access_loggers.stdout
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                    log_format:
                      text_format_source:
                        inline_string: "access request_id=%REQ(X-REQUEST-ID)% listener={name} method=%REQ(:METHOD)% path=%REQ(:PATH)% code=%RESPONSE_CODE% flags=%RESPONSE_FLAGS%\\n"
              route_config:
                name: route-{name}
                virtual_hosts:
                  - name: local-{name}
                    domains: ["*"]
                    routes:
                      - match: {{prefix: "/"}}
                        route: {{cluster: {cluster}, timeout: 30s}}
              http_filters:
                - name: envoy.filters.http.wasm
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                    config:
                      name: {name}
                      root_id: {name}
                      vm_config:
                        vm_id: {name}
                        runtime: envoy.wasm.runtime.v8
                        code:
                          local: {{filename: /runtime/{wasm_file}}}
                      configuration:
                        "@type": type.googleapis.com/google.protobuf.StringValue
                        value: '{config_json}'
                - name: envoy.filters.http.router
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
'''


def cluster_yaml(name):
    return f'''  - name: {name}
    connect_timeout: 5s
    type: STRICT_DNS
    dns_lookup_family: V4_ONLY
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: {name}
      endpoints:
        - lb_endpoints:
            - endpoint:
                address:
                  socket_address: {{address: {name}, port_value: 8080}}
'''


def single_listener_config(admin_port, listener_port, name, config, wasm_file="plugin.wasm"):
    rendered = f'''admin:
  address:
    socket_address: {{address: 0.0.0.0, port_value: {admin_port}}}
static_resources:
  listeners:
'''
    rendered += listener_yaml(listener_port, name, "backend-primary", config, wasm_file)
    rendered += "  clusters:\n" + cluster_yaml("backend-primary")
    return rendered


def lds_discovery_response(version, config):
    listener = listener_yaml(12008, "schema-compatibility-generation", "backend-primary", config)
    listener = listener.replace(
        "  - name:",
        '  - "@type": type.googleapis.com/envoy.config.listener.v3.Listener\n    name:',
        1,
    )
    return f'''version_info: "{version}"
resources:
{listener}
type_url: type.googleapis.com/envoy.config.listener.v3.Listener
'''


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    main_config = '''admin:
  address:
    socket_address: {address: 0.0.0.0, port_value: 9901}
static_resources:
  listeners:
'''
    main_config += "".join(listener_yaml(*entry) for entry in LISTENERS)
    main_config += "  clusters:\n" + cluster_yaml("backend-primary") + cluster_yaml("backend-secondary")
    (OUT / "envoy.yaml").write_text(main_config)

    auto = proxy("proxy-auto-rejected", "auto")
    auto_config = '''admin:
  address:
    socket_address: {address: 0.0.0.0, port_value: 9911}
static_resources:
  listeners:
'''
    auto_config += listener_yaml(11000, "proxy-auto-rejected", "backend-primary", auto)
    auto_config += "  clusters:\n" + cluster_yaml("backend-primary")
    (OUT / "envoy-auto.yaml").write_text(auto_config)

    (OUT / "envoy-baseline.yaml").write_text(single_listener_config(
        9931, 13008, "schema-compatibility-baseline", SCHEMA_COMPATIBILITY, "baseline-plugin.wasm",
    ))
    (OUT / "envoy-oracle.yaml").write_text(single_listener_config(
        9941, 13018, "schema-compatibility-oracle", SCHEMA_COMPATIBILITY, "oracle-plugin.wasm",
    ))
    for suffix, wasm_file, admin_port, listener_port in (
        ("candidate", "plugin.wasm", 9951, 13108),
        ("affected", "baseline-plugin.wasm", 9961, 13118),
        ("oracle", "oracle-plugin.wasm", 9971, 13128),
    ):
        (OUT / f"envoy-control-{suffix}.yaml").write_text(single_listener_config(
            admin_port, listener_port, f"malformed-control-{suffix}", MALFORMED_NON_SCHEMA, wasm_file,
        ))
    generation_configs = (
        ("valid-before", VALID_SCHEMA_COMPATIBILITY),
        ("validation-unavailable", SCHEMA_COMPATIBILITY),
        ("valid-after", VALID_SCHEMA_COMPATIBILITY),
    )
    for phase, config in generation_configs:
        (OUT / f"lds-generation-{phase}.yaml").write_text(lds_discovery_response(phase, config))
    (OUT / "lds-generation-current.yaml").write_text(lds_discovery_response(*generation_configs[0]))
    generation_bootstrap = '''admin:
  address:
    socket_address: {address: 0.0.0.0, port_value: 9921}
node:
  id: schema-compatibility-generation
  cluster: runtime-verification
dynamic_resources:
  lds_config:
    path_config_source:
      path: /evidence/lds-generation-current.yaml
    resource_api_version: V3
static_resources:
  clusters:
'''
    generation_bootstrap += cluster_yaml("backend-primary")
    (OUT / "envoy-generation.yaml").write_text(generation_bootstrap)


if __name__ == "__main__":
    main()

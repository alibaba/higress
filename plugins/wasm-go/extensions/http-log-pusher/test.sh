NAMESPACE="himarket-system"
# 1. 获取 higress-gateway Pod
POD=$(kubectl get pod -n "$NAMESPACE" -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')

kubectl logs -n "$NAMESPACE" "$POD" -c higress-gateway --tail=100 -f --timestamps | grep --color -E "^|$|ERROR|WARN|404|500|http-log-pusher|error|collector_service"


# kubectl get wasmplugin http-log-push-plugin -n "$NAMESPACE" -o yaml > http-log-push-plugin.yaml

# 2. 检查 travel-assistant 路由是否加载了 wasm filter
# kubectl exec -n ls-test $POD -c higress-gateway -- \
#   curl -s localhost:15000/config_dump | \
#   jq -r '.configs[] | select(.name=="routes") | .route_config.virtual_hosts[].routes[] | 
#     select(.match.prefix=="/mcp-servers/travel-assistant") | 
#     .route.typed_per_filter_config | keys[]' | grep wasm

# kubectl exec -n ls-test $POD -c higress-gateway -- \
#   curl -s localhost:15000/config_dump | \
#   jq -r '
#     .configs[] 
#     | select(.["@type"] == "type.googleapis.com/envoy.config.route.v3.RouteConfiguration") 
#     | .virtual_hosts[] 
#     | .routes[] 
#     | select(.match.prefix == "/mcp-servers/travel-assistant") 
#     | .route.typed_per_filter_config 
#     | to_entries[] 
#     | select(.key | contains("wasm")) 
#     | "PLUGIN: \(.key)\nCONFIG: \(.value | tojson)"
#   '

# POD=$(kubectl get pod -n ls-test -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')

# # 查找所有包含 "travel-assistant" 的路由（不严格匹配路径）
# kubectl exec -n ls-test $POD -c higress-gateway -- \
#   curl -s localhost:15000/config_dump > config_dump.json

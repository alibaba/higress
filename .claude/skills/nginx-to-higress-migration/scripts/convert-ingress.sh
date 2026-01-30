#!/bin/bash
# Convert nginx Ingress annotations to Higress format (OPTIONAL)
#
# NOTE: Higress natively supports nginx.ingress.kubernetes.io/* annotations!
# This script is only needed if you want to:
# 1. Standardize on higress.io/* prefix
# 2. Identify unsupported annotations that need WASM plugins

set -e

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <namespace> <ingress-name> [output-file]"
    echo ""
    echo "Example: $0 default my-app-ingress higress-ingress.yaml"
    echo ""
    echo "NOTE: This conversion is OPTIONAL - Higress supports nginx annotations natively!"
    exit 1
fi

NAMESPACE="$1"
INGRESS_NAME="$2"
OUTPUT_FILE="${3:-higress-${INGRESS_NAME}.yaml}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Converting Ingress: ${NAMESPACE}/${INGRESS_NAME}${NC}"

# Get original Ingress
ORIGINAL=$(kubectl get ingress "$INGRESS_NAME" -n "$NAMESPACE" -o json 2>/dev/null)

if [ -z "$ORIGINAL" ]; then
    echo -e "${RED}Error: Ingress ${NAMESPACE}/${INGRESS_NAME} not found${NC}"
    exit 1
fi

# Annotation mapping
declare -A ANNOTATION_MAP=(
    ["nginx.ingress.kubernetes.io/rewrite-target"]="higress.io/rewrite-target"
    ["nginx.ingress.kubernetes.io/use-regex"]="higress.io/use-regex"
    ["nginx.ingress.kubernetes.io/ssl-redirect"]="higress.io/ssl-redirect"
    ["nginx.ingress.kubernetes.io/force-ssl-redirect"]="higress.io/force-ssl-redirect"
    ["nginx.ingress.kubernetes.io/backend-protocol"]="higress.io/backend-protocol"
    ["nginx.ingress.kubernetes.io/proxy-body-size"]="higress.io/proxy-body-size"
    ["nginx.ingress.kubernetes.io/enable-cors"]="higress.io/enable-cors"
    ["nginx.ingress.kubernetes.io/cors-allow-origin"]="higress.io/cors-allow-origin"
    ["nginx.ingress.kubernetes.io/cors-allow-methods"]="higress.io/cors-allow-methods"
    ["nginx.ingress.kubernetes.io/cors-allow-headers"]="higress.io/cors-allow-headers"
    ["nginx.ingress.kubernetes.io/cors-expose-headers"]="higress.io/cors-expose-headers"
    ["nginx.ingress.kubernetes.io/cors-allow-credentials"]="higress.io/cors-allow-credentials"
    ["nginx.ingress.kubernetes.io/cors-max-age"]="higress.io/cors-max-age"
    ["nginx.ingress.kubernetes.io/proxy-connect-timeout"]="higress.io/proxy-connect-timeout"
    ["nginx.ingress.kubernetes.io/proxy-send-timeout"]="higress.io/proxy-send-timeout"
    ["nginx.ingress.kubernetes.io/proxy-read-timeout"]="higress.io/proxy-read-timeout"
    ["nginx.ingress.kubernetes.io/proxy-next-upstream-tries"]="higress.io/proxy-next-upstream-tries"
    ["nginx.ingress.kubernetes.io/canary"]="higress.io/canary"
    ["nginx.ingress.kubernetes.io/canary-weight"]="higress.io/canary-weight"
    ["nginx.ingress.kubernetes.io/canary-header"]="higress.io/canary-header"
    ["nginx.ingress.kubernetes.io/canary-header-value"]="higress.io/canary-header-value"
    ["nginx.ingress.kubernetes.io/canary-header-pattern"]="higress.io/canary-header-pattern"
    ["nginx.ingress.kubernetes.io/canary-by-cookie"]="higress.io/canary-by-cookie"
    ["nginx.ingress.kubernetes.io/auth-type"]="higress.io/auth-type"
    ["nginx.ingress.kubernetes.io/auth-secret"]="higress.io/auth-secret"
    ["nginx.ingress.kubernetes.io/auth-realm"]="higress.io/auth-realm"
    ["nginx.ingress.kubernetes.io/load-balance"]="higress.io/load-balance"
    ["nginx.ingress.kubernetes.io/upstream-hash-by"]="higress.io/upstream-hash-by"
    ["nginx.ingress.kubernetes.io/whitelist-source-range"]="higress.io/whitelist-source-range"
    ["nginx.ingress.kubernetes.io/denylist-source-range"]="higress.io/denylist-source-range"
    ["nginx.ingress.kubernetes.io/permanent-redirect"]="higress.io/permanent-redirect"
    ["nginx.ingress.kubernetes.io/temporal-redirect"]="higress.io/temporal-redirect"
)

# Unsupported annotations (will be logged as warnings)
UNSUPPORTED_PATTERNS="server-snippet|configuration-snippet|stream-snippet|lua-resty|modsecurity|limit-rps|limit-connections"

# Build converted annotations
echo -e "${YELLOW}Converting annotations...${NC}"

WARNINGS=()
CONVERTED_ANNOTATIONS="{}"

# Get original annotations
ORIG_ANNOTATIONS=$(echo "$ORIGINAL" | jq '.metadata.annotations // {}')

# Convert each annotation
for key in $(echo "$ORIG_ANNOTATIONS" | jq -r 'keys[]'); do
    value=$(echo "$ORIG_ANNOTATIONS" | jq -r --arg k "$key" '.[$k]')
    
    # Check if it's a nginx annotation
    if [[ "$key" == nginx.ingress.kubernetes.io/* ]]; then
        # Check if unsupported
        anno_name=$(echo "$key" | sed 's/nginx.ingress.kubernetes.io\///')
        if echo "$anno_name" | grep -qE "$UNSUPPORTED_PATTERNS"; then
            WARNINGS+=("$key (requires WASM plugin)")
            continue
        fi
        
        # Check if we have a mapping
        if [ -n "${ANNOTATION_MAP[$key]}" ]; then
            new_key="${ANNOTATION_MAP[$key]}"
            CONVERTED_ANNOTATIONS=$(echo "$CONVERTED_ANNOTATIONS" | jq --arg k "$new_key" --arg v "$value" '. + {($k): $v}')
            echo -e "  ${GREEN}✓${NC} $key → $new_key"
        else
            # Keep as-is if no mapping (might still work)
            CONVERTED_ANNOTATIONS=$(echo "$CONVERTED_ANNOTATIONS" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
            echo -e "  ${YELLOW}?${NC} $key (kept as-is)"
        fi
    elif [[ "$key" == "kubernetes.io/ingress.class" ]]; then
        # Update ingress class annotation
        CONVERTED_ANNOTATIONS=$(echo "$CONVERTED_ANNOTATIONS" | jq '. + {"kubernetes.io/ingress.class": "higress"}')
        echo -e "  ${GREEN}✓${NC} Updated ingress.class to higress"
    else
        # Keep other annotations
        CONVERTED_ANNOTATIONS=$(echo "$CONVERTED_ANNOTATIONS" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
    fi
done

# Add migration marker
CONVERTED_ANNOTATIONS=$(echo "$CONVERTED_ANNOTATIONS" | jq '. + {"higress.io/migrated-from": "nginx"}')

# Build new Ingress
echo -e "\n${YELLOW}Generating Higress Ingress...${NC}"

NEW_INGRESS=$(echo "$ORIGINAL" | jq --argjson anno "$CONVERTED_ANNOTATIONS" '
    .metadata.annotations = $anno |
    .spec.ingressClassName = "higress" |
    del(.metadata.resourceVersion) |
    del(.metadata.uid) |
    del(.metadata.creationTimestamp) |
    del(.metadata.generation) |
    del(.metadata.managedFields) |
    del(.status)
')

# Output
echo "$NEW_INGRESS" | yq -P > "$OUTPUT_FILE"

echo -e "\n${GREEN}✓ Converted Ingress saved to: ${OUTPUT_FILE}${NC}"

# Print warnings
if [ ${#WARNINGS[@]} -gt 0 ]; then
    echo -e "\n${YELLOW}⚠ Unsupported annotations detected:${NC}"
    for warning in "${WARNINGS[@]}"; do
        echo -e "  ${RED}•${NC} $warning"
    done
    echo -e "\nThese features require WASM plugins. See the skill documentation for conversion patterns."
fi

echo -e "\n${YELLOW}Next steps:${NC}"
echo "1. Review the generated file: $OUTPUT_FILE"
echo "2. Ensure Higress is installed: helm install higress higress/higress -n higress-system"
echo "3. Apply the converted Ingress: kubectl apply -f $OUTPUT_FILE"
echo "4. Verify: kubectl get ingress -n $NAMESPACE"

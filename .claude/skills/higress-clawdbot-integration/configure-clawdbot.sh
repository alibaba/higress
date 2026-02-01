#!/usr/bin/env bash

# Clawdbot Integration Configuration Script
# This script configures Higress AI Gateway for Clawdbot/OpenClaw integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if Clawdbot is installed
checkClawdbot() {
  if [ -d "$HOME/clawd" ]; then
    return 0
  elif command -v clawdbot &> /dev/null; then
    return 0
  else
    return 1
  fi
}

# Function to configure Clawdbot integration
configureClawdbotIntegration() {
  if ! checkClawdbot; then
    echo -e "${YELLOW}Warning: Clawdbot not detected. Skipping integration setup.${NC}"
    echo "If you want to use Clawdbot with Higress AI Gateway, install it first:"
    echo "  curl -fsSL https://clawd.bot/install.sh | bash"
    return 0
  fi

  # Determine Clawdbot workspace
  if [ -d "$HOME/clawd" ]; then
    CLAWDBOT_WORKSPACE="$HOME/clawd"
  else
    CLAWDBOT_WORKSPACE=$(command -v clawdbot | xargs dirname)
  fi

  echo
  echo "======================================================="
  echo "          Clawdbot Integration Detected                "
  echo "======================================================="
  echo
  echo "Clawdbot workspace found at: $CLAWDBOT_WORKSPACE"
  echo
  echo "Higress AI Gateway can integrate with Clawdbot to provide:"
  echo "  1. Auto-routing: Automatically route requests to different models"
  echo "     based on message content (e.g., 'deep thinking' → claude-opus-4.5)"
  echo "  2. Model provider: Use Higress as a unified model provider in Clawdbot"
  echo

  # Ask about auto-routing
  read -p "Enable auto-routing feature? (y/N): " enableAutoRouting
  case "$enableAutoRouting" in
    [yY]|[yY][eE][sS])
      ENABLE_AUTO_ROUTING="true"
      
      echo
      echo "Auto-routing allows you to route requests to different models based on"
      echo "keywords in your message. For example:"
      echo "  - '深入思考 ...' or 'deep thinking ...' → reasoning model"
      echo "  - '写代码 ...' or 'code: ...' → coding model"
      echo

      # Get default model for auto-routing
      read -p "Default model when no routing rule matches (default: qwen-turbo): " defaultModel
      if [ -z "$defaultModel" ]; then
        AUTO_ROUTING_DEFAULT_MODEL="qwen-turbo"
      else
        AUTO_ROUTING_DEFAULT_MODEL="$defaultModel"
      fi

      echo
      echo "You can configure routing rules later using natural language in Clawdbot."
      echo "For example, say: 'route to claude-opus-4.5 when solving difficult problems'"
      echo
      ;;
    *)
      ENABLE_AUTO_ROUTING="false"
      echo "Auto-routing disabled. You can enable it later via Higress Console."
      ;;
  esac

  # Return configuration as environment variables
  if [ "$ENABLE_AUTO_ROUTING" = "true" ]; then
    echo "export ENABLE_AUTO_ROUTING=true"
    echo "export AUTO_ROUTING_DEFAULT_MODEL=$AUTO_ROUTING_DEFAULT_MODEL"
  fi
}

# Function to apply auto-routing configuration to Higress
applyAutoRoutingConfig() {
  if [ "$ENABLE_AUTO_ROUTING" != "true" ]; then
    return 0
  fi

  echo "Configuring auto-routing in Higress AI Gateway..."

  # Check if Higress container is running
  if ! docker ps | grep -q "higress-ai-gateway"; then
    echo -e "${RED}Error: Higress AI Gateway container is not running${NC}"
    echo "Please start it first using get-ai-gateway.sh start"
    return 1
  fi

  local MODEL_ROUTER_FILE="./higress/wasmplugins/model-router.internal.yaml"
  local CONTAINER_MODEL_ROUTER_FILE="/data/wasmplugins/model-router.internal.yaml"

  # Wait for file to be created
  local MAX_WAIT=30
  local WAIT_COUNT=0
  while [ ! -f "$MODEL_ROUTER_FILE" ] && [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
  done

  if [ ! -f "$MODEL_ROUTER_FILE" ]; then
    echo -e "${YELLOW}Warning: Could not find model-router configuration file${NC}"
    echo "Auto-routing will be configured manually later."
    return 1
  fi

  # Update model-router configuration
  docker exec -i -e DEFAULT_MODEL="$AUTO_ROUTING_DEFAULT_MODEL" -e MODEL_ROUTER_FILE="$CONTAINER_MODEL_ROUTER_FILE" higress-ai-gateway /bin/sh <<'EOF'
set -e
cp ${MODEL_ROUTER_FILE} ${MODEL_ROUTER_FILE}.backup
awk -v model="$DEFAULT_MODEL" '
  /modelToHeader: x-higress-llm-model/ {
    print
    print "    autoRouting:"
    print "      enable: true"
    print "      defaultModel: " model
    next
  }
  { print }
' ${MODEL_ROUTER_FILE} > /tmp/model-router.internal.yaml.tmp.$$
mv /tmp/model-router.internal.yaml.tmp.* ${MODEL_ROUTER_FILE}
EOF

  echo -e "${GREEN}✓ Auto-routing configured with default model: $AUTO_ROUTING_DEFAULT_MODEL${NC}"
  echo "  Configuration file: $MODEL_ROUTER_FILE"
}

# Function to configure Clawdbot to use Higress
configureClawdbotProvider() {
  echo "Configuring Clawdbot to use Higress AI Gateway..."
  echo

  if command -v clawdbot &> /dev/null; then
    echo "Running: clawdbot models auth login --provider higress"
    clawdbot models auth login --provider higress
  elif command -v openclaw &> /dev/null; then
    echo "Running: openclaw models auth login --provider higress"
    openclaw models auth login --provider higress
  else
    echo -e "${RED}Error: Neither clawdbot nor openclaw command found${NC}"
    return 1
  fi

  echo -e "${GREEN}✓ Clawdbot configured to use Higress AI Gateway${NC}"
}

# Main execution
if [ "${1:-}" = "configure" ]; then
  configureClawdbotIntegration
elif [ "${1:-}" = "apply" ]; then
  applyAutoRoutingConfig
elif [ "${1:-}" = "setup" ]; then
  configureClawdbotIntegration
  applyAutoRoutingConfig
  configureClawdbotProvider
else
  echo "Usage: $0 {configure|apply|setup}"
  echo ""
  echo "Commands:"
  echo "  configure  - Ask for Clawdbot integration configuration"
  echo "  apply      - Apply auto-routing configuration to Higress"
  echo "  setup      - Full setup (configure + apply + provider setup)"
  exit 1
fi

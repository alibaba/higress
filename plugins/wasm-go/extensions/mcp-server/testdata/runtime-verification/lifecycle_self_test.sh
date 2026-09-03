#!/usr/bin/env bash
set -u

HARNESS_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$HARNESS_DIR/lifecycle.sh"

fail() {
  echo "lifecycle self-test failed: $1" >&2
  exit 1
}

runtime_sleep() { :; }
compose_status=0
service_status=1
compose_runtime() {
  test "$1" = stop || return 2
  return "$compose_status"
}
runtime_service_running() { return "$service_status"; }

compose_status=1
service_status=1
RUNTIME_STOP_ATTEMPTS=1 stop_runtime_service stopped-after-error || \
  fail "non-zero stop was rejected despite confirmed stopped state"

compose_status=1
service_status=0
if RUNTIME_STOP_ATTEMPTS=1 stop_runtime_service still-running; then
  fail "non-zero stop was accepted while the container remained running"
fi

compose_status=0
service_status=0
if RUNTIME_STOP_ATTEMPTS=1 stop_runtime_service success-but-running; then
  fail "successful stop command bypassed the running-state inspection"
fi

health_attempt=0
reset_attempt=0
backend_health_probe() {
  health_attempt=$((health_attempt + 1))
  test "$health_attempt" -ge 2
}
backend_reset_probe() {
  reset_attempt=$((reset_attempt + 1))
  test "$reset_attempt" -ge 2
}
RUNTIME_BACKEND_HEALTH_ATTEMPTS=3 RUNTIME_BACKEND_RESET_ATTEMPTS=3 reset_primary_backend || \
  fail "transient backend health/reset failures did not recover"
test "$health_attempt" -eq 2 || fail "health retry count changed"
test "$reset_attempt" -eq 2 || fail "reset retry count changed"

backend_health_probe() { return 1; }
if RUNTIME_BACKEND_HEALTH_ATTEMPTS=2 reset_primary_backend; then
  fail "permanently unavailable backend was accepted"
fi

backend_health_probe() { return 0; }
backend_reset_probe() { return 1; }
if RUNTIME_BACKEND_RESET_ATTEMPTS=2 reset_primary_backend; then
  fail "permanent reset failure was accepted"
fi

echo "runtime lifecycle fault-injection self-tests passed"

#!/usr/bin/env bash
# Shared bounded lifecycle gates. The caller provides compose_runtime().

runtime_sleep() {
  sleep "$1"
}

runtime_diagnostic() {
  message=$1
  echo "$message" >&2
  if test -n "${RUNTIME_LIFECYCLE_DIAGNOSTICS:-}"; then
    echo "$message" >>"$RUNTIME_LIFECYCLE_DIAGNOSTICS"
  fi
}

runtime_service_running() {
  service=$1
  container_id=$(compose_runtime ps -a -q "$service" 2>/dev/null) || return 2
  test -n "$container_id" || return 2
  running=$(podman inspect "$container_id" --format '{{.State.Running}}' 2>/dev/null) || return 2
  case "$running" in
    true) return 0 ;;
    false) return 1 ;;
    *) return 2 ;;
  esac
}

stop_runtime_service() {
  service=$1
  attempts=${RUNTIME_STOP_ATTEMPTS:-20}
  stop_status=0
  compose_runtime stop -t 10 "$service" >/dev/null 2>&1 || stop_status=$?
  attempt=1
  while test "$attempt" -le "$attempts"; do
    running_status=0
    runtime_service_running "$service" || running_status=$?
    if test "$running_status" -eq 1; then
      return 0
    fi
    runtime_sleep 0.5
    attempt=$((attempt + 1))
  done
  runtime_diagnostic "failed to confirm $service stopped after compose status $stop_status and $attempts inspections"
  return 1
}

backend_health_probe() {
  compose_runtime exec -T backend-primary wget -q -O /dev/null http://127.0.0.1:8080/healthz
}

backend_reset_probe() {
  compose_runtime exec -T backend-primary wget -q -O /dev/null \
    --post-data '{}' http://127.0.0.1:8080/__reset
}

wait_primary_backend_ready() {
  attempts=${RUNTIME_BACKEND_HEALTH_ATTEMPTS:-60}
  attempt=1
  while test "$attempt" -le "$attempts"; do
    if backend_health_probe; then
      return 0
    fi
    runtime_sleep 0.5
    attempt=$((attempt + 1))
  done
  runtime_diagnostic "backend-primary remained unavailable after $attempts health probes"
  return 1
}

reset_primary_backend() {
  wait_primary_backend_ready || return 1
  attempts=${RUNTIME_BACKEND_RESET_ATTEMPTS:-20}
  attempt=1
  while test "$attempt" -le "$attempts"; do
    if backend_reset_probe; then
      return 0
    fi
    runtime_sleep 0.5
    attempt=$((attempt + 1))
  done
  runtime_diagnostic "backend-primary was healthy but reset failed after $attempts attempts"
  return 1
}

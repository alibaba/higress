#!/usr/bin/env python3
"""Finalize sanitized, machine-readable runtime evidence."""

import hashlib
import json
import os
import re
from collections import Counter
from pathlib import Path


root = Path(os.environ["RUNTIME_EVIDENCE"])
matrix_path = root / "matrix.json"
matrix = json.loads(matrix_path.read_text()) if matrix_path.exists() else {"cases": []}

baseline_log = (root / "gateway-baseline.log").read_text(errors="replace") if (root / "gateway-baseline.log").exists() else ""
baseline_state = json.loads((root / "backend-baseline-state.json").read_text()) if (root / "backend-baseline-state.json").exists() else {"events": ["missing"]}
baseline_ok = (
    "requires a primitive type" in baseline_log
    and "plugin start failed" in baseline_log
    and baseline_state.get("events") == []
)
matrix["cases"].append({
    "case": "schema-compatibility-pinned-baseline-is-rejected",
    "status": "PASS" if baseline_ok else "FAIL",
    "detail": {
        "sourceSha": os.environ["BASELINE_SHA"],
        "pluginSha256": os.environ["BASELINE_PLUGIN_SHA256"],
        "configurationAccepted": False,
        "compilerRejection": "enum requires a primitive type",
        "upstreamCalls": 0,
        "backendEvents": {"backend-primary": []},
    } if baseline_ok else {"error": "expected pinned baseline compiler rejection or zero-upstream proof is absent"},
})

oracle_path = root / "oracle-verification.json"
oracle = json.loads(oracle_path.read_text()) if oracle_path.exists() else {}
oracle_log = (root / "gateway-oracle.log").read_text(errors="replace") if (root / "gateway-oracle.log").exists() else ""
oracle_events = oracle.get("backendEvents", {}).get("backend-primary", [])
oracle_ok = (
    oracle.get("configurationAccepted") is True
    and oracle.get("descriptorPreserved") is True
    and oracle.get("legacyRESTMapping") is True
    and len(oracle_events) == 1
    and "plugin start failed" not in oracle_log
)
matrix["cases"].append({
    "case": "schema-compatibility-v2.0.0-oracle-accepts-and-invokes",
    "status": "PASS" if oracle_ok else "FAIL",
    "detail": {
        "sourceSha": os.environ["ORACLE_SHA"],
        "pluginSha256": os.environ["ORACLE_PLUGIN_SHA256"],
        "configurationAccepted": True,
        "descriptorPreserved": True,
        "legacyRESTMapping": True,
        "upstreamCalls": len(oracle_events),
        "backendEvents": {"backend-primary": oracle_events},
        "oracleEvidence": "oracle-verification.json",
    } if oracle_ok else {"error": "v2.0.0 compatibility oracle acceptance, descriptor, or REST mapping proof is absent"},
})

control_state = json.loads((root / "backend-control-state.json").read_text()) if (root / "backend-control-state.json").exists() else {"events": ["missing"]}
control_logs = {
    variant: (root / f"gateway-control-{variant}.log").read_text(errors="replace")
    if (root / f"gateway-control-{variant}.log").exists() else ""
    for variant in ("oracle", "affected", "candidate")
}
control_rejections = {
    variant: "error parsing URL template" in log and "plugin start failed" in log
    for variant, log in control_logs.items()
}
control_ok = all(control_rejections.values()) and control_state.get("events") == []
matrix["cases"].append({
    "case": "malformed-non-schema-control-is-rejected-by-all-revisions",
    "status": "PASS" if control_ok else "FAIL",
    "detail": {
        "revisions": {
            "oracle": os.environ["ORACLE_SHA"],
            "affected": os.environ["BASELINE_SHA"],
            "candidate": os.environ["SOURCE_SHA"],
        },
        "configurationAccepted": {variant: False for variant in control_rejections},
        "rejection": "error parsing URL template",
        "upstreamCalls": 0,
        "backendEvents": {"backend-primary": []},
    } if control_ok else {"error": f"non-schema rejection proof is incomplete: {control_rejections}"},
})

generation_phases = ("valid-before", "validation-unavailable", "valid-after")
generation_path = root / "generation-transition.json"
generation_transition = json.loads(generation_path.read_text()) if generation_path.exists() else {"generations": []}
generation_records = generation_transition.get("generations", [])
process_before = (root / "generation-process-before.txt").read_text().strip() if (root / "generation-process-before.txt").exists() else "missing-before"
process_after = (root / "generation-process-after.txt").read_text().strip() if (root / "generation-process-after.txt").exists() else "missing-after"
generation_ok = (
    [record.get("phase") for record in generation_records] == list(generation_phases)
    and [record.get("schemaState") for record in generation_records] == ["validated", "validation-unavailable", "validated"]
    and [len(record.get("backendEvents", {}).get("backend-primary", [])) for record in generation_records] == [1, 0, 1]
    and generation_transition.get("ldsVersions") == list(generation_phases)
    and process_before == process_after
    and not process_before.startswith("missing")
)
matrix["cases"].append({
    "case": "schema-compatibility-same-process-dynamic-generation-transition",
    "status": "PASS" if generation_ok else "FAIL",
    "detail": {
        "sequence": [record.get("schemaState") for record in generation_records],
        "backendCallsByGeneration": [len(record["backendEvents"]["backend-primary"]) for record in generation_records],
        "runtimeBoundary": "same Envoy process and Wasm, file-backed LDS config generations",
        "processIdentity": process_before,
        "generationEvidence": "generation-transition.json",
    } if generation_ok else {"error": "same-process valid -> validation-unavailable -> valid LDS evidence is incomplete"},
})

auto_log = (root / "gateway-auto.log").read_text(errors="replace") if (root / "gateway-auto.log").exists() else ""
auto_state = json.loads((root / "backend-auto-state.json").read_text()) if (root / "backend-auto-state.json").exists() else {"events": ["missing"]}
auto_ok = "invalid protocolStrategy value: auto" in auto_log and auto_state.get("events") == []
matrix["cases"].append({
    "case": "proxy-auto-configuration-is-rejected",
    "status": "PASS" if auto_ok else "FAIL",
    "detail": {"configuredValue": "auto", "accepted": False, "upstreamCalls": 0, "backendEvents": {"backend-primary": []}}
    if auto_ok else {"error": "expected configuration rejection absent from sanitized gateway log"},
})
matrix["summary"] = {
    "pass": sum(case["status"] == "PASS" for case in matrix["cases"]),
    "fail": sum(case["status"] != "PASS" for case in matrix["cases"]),
}
matrix_path.write_text(json.dumps(matrix, indent=2, sort_keys=True) + "\n")

client_exchanges = json.loads((root / "client-exchanges.json").read_text()).get("exchanges", [])
expected_request_ids = [exchange.get("accessRequestId") for exchange in client_exchanges]
gateway_log = (root / "gateway.log").read_text(errors="replace") if (root / "gateway.log").exists() else ""
access_request_ids = re.findall(r"access request_id=([^ ]+)", gateway_log)
expected_counts = Counter(expected_request_ids)
access_counts = Counter(access_request_ids)
missing_request_ids = sorted(request_id for request_id in expected_counts if access_counts[request_id] == 0)
duplicate_request_ids = sorted(request_id for request_id, count in access_counts.items() if count != 1)
unexpected_request_ids = sorted(request_id for request_id in access_counts if request_id not in expected_counts)
access_ok = (
    len(expected_request_ids) == len(set(expected_request_ids))
    and len(access_request_ids) == len(expected_request_ids)
    and not missing_request_ids
    and not duplicate_request_ids
    and not unexpected_request_ids
)
access_coverage = {
    "status": "PASS" if access_ok else "FAIL",
    "recordedClientExchangeCount": len(expected_request_ids),
    "accessRecordCount": len(access_request_ids),
    "missingRequestIds": missing_request_ids,
    "duplicateRequestIds": duplicate_request_ids,
    "unexpectedRequestIds": unexpected_request_ids,
}
(root / "access-coverage.json").write_text(json.dumps(access_coverage, indent=2, sort_keys=True) + "\n")


def lines(name):
    path = root / name
    return path.read_text(errors="replace").strip().splitlines() if path.exists() else []


manifest = {
    "source_sha": os.environ["SOURCE_SHA"],
    "source_tree_clean": os.environ.get("SOURCE_TREE_CLEAN") == "true",
    "plugin_sha256": os.environ["PLUGIN_SHA256"],
    "baseline_source_sha": os.environ["BASELINE_SHA"],
    "baseline_plugin_sha256": os.environ["BASELINE_PLUGIN_SHA256"],
    "oracle_source_sha": os.environ["ORACLE_SHA"],
    "oracle_plugin_sha256": os.environ["ORACLE_PLUGIN_SHA256"],
    "gateway_image": "higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway:v2.2.3",
    "gateway_resolved_digests": lines("gateway-image-digests.txt"),
    "backend_image": "docker.io/library/python:3.12-alpine",
    "backend_resolved_digests": lines("backend-image-digests.txt"),
    "podman_version": (lines("podman-version.txt") or [""])[0],
    "compose_version": lines("compose-version.txt"),
    "matrix": "matrix.json",
    "client_exchange_count": len(client_exchanges),
    "access_coverage": access_coverage,
    "evidence_index": [
        "manifest.json", "matrix.json", "client-exchanges.json", "access-coverage.json", "compose-config.yaml", "envoy.yaml", "envoy-auto.yaml",
        "envoy-baseline.yaml", "envoy-oracle.yaml", "envoy-control-candidate.yaml",
        "envoy-control-affected.yaml", "envoy-control-oracle.yaml", "envoy-generation.yaml", "lds-generation-valid-before.yaml",
        "lds-generation-validation-unavailable.yaml", "lds-generation-valid-after.yaml", "lds-generation-current.yaml",
        "gateway.log", "gateway-auto.log", "gateway-baseline.log", "gateway-oracle.log",
        "gateway-control-candidate.log", "gateway-control-affected.log", "gateway-control-oracle.log", "gateway-generation.log",
        "backend-auto-state.json", "backend-baseline-state.json", "backend-oracle-state.json", "backend-control-state.json",
        "oracle-verification.json", "generation-transition.json",
        "generation-process-before.txt", "generation-process-after.txt",
        "backend-primary-final.json", "backend-secondary-final.json",
        "cleanup-proof.txt", "SHA256SUMS",
    ],
    "sanitization": "fake credentials are redacted from textual logs; backend evidence stores only presence/match booleans and never raw credentials or session IDs",
    "cleanup": (lines("cleanup-proof.txt") or ["pending"])[0],
}
(root / "manifest.json").write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")

checksums = []
for path in sorted(root.iterdir()):
    if path.is_file() and path.name not in ("plugin.wasm", "baseline-plugin.wasm", "oracle-plugin.wasm", "SHA256SUMS"):
        checksums.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}")
(root / "SHA256SUMS").write_text("\n".join(checksums) + "\n")

print(json.dumps({"evidence": str(root), "source_sha": manifest["source_sha"], "plugin_sha256": manifest["plugin_sha256"], "access_coverage": access_coverage["status"], **matrix["summary"]}, sort_keys=True))
raise SystemExit(0 if matrix["summary"]["fail"] == 0 and access_ok else 1)

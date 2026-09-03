#!/usr/bin/env python3
"""Focused fault-injection checks for runtime lifecycle polling and fixtures."""

import json
import socket
import tempfile
import threading
import urllib.request
from pathlib import Path

import backend
import verify


def check_transient_lds_publish_timeout():
    attempts = iter((socket.timeout("transient"), socket.timeout("transient"), None))
    original = verify.exchange

    def fake_exchange(_url, **_kwargs):
        outcome = next(attempts)
        if outcome is not None:
            raise outcome
        return 200, {}, {"configs": [{"version_info": "candidate-fixture"}]}

    verify.exchange = fake_exchange
    try:
        verify.wait_lds_version("candidate-fixture")
    finally:
        verify.exchange = original


def check_rejection_poll_classification():
    original = verify.read_lds_rejected_count
    attempts = iter((socket.timeout("transient"), 4))

    def transient_then_rejected(_admin_port):
        outcome = next(attempts)
        if isinstance(outcome, Exception):
            raise outcome
        return outcome

    verify.read_lds_rejected_count = transient_then_rejected
    try:
        assert verify.wait_lds_rejected_count(greater_than=3) == 4
    finally:
        verify.read_lds_rejected_count = original

    verify.read_lds_rejected_count = lambda _admin_port: (_ for _ in ()).throw(socket.timeout("down"))
    try:
        try:
            verify.wait_lds_rejected_count(timeout=0.01)
        except RuntimeError as exc:
            assert "checker remained unavailable" in str(exc)
        else:
            raise AssertionError("permanent admin failure was accepted as an LDS rejection")
    finally:
        verify.read_lds_rejected_count = original

    verify.read_lds_rejected_count = lambda _admin_port: 3
    try:
        try:
            verify.wait_lds_rejected_count(greater_than=3, timeout=0.01)
        except RuntimeError as exc:
            assert "did not report an LDS rejection" in str(exc)
        else:
            raise AssertionError("published stats without a rejection increment were accepted")
    finally:
        verify.read_lds_rejected_count = original


def check_corpus_get_fixture():
    with backend.LOCK:
        backend.EVENTS = []
        backend.SEQ = 0
    # BaseHTTPServer performs a host-name lookup during bind; isolate this unit
    # check from developer-machine DNS state.
    original_getfqdn = socket.getfqdn
    socket.getfqdn = lambda _name: "localhost"
    try:
        server = backend.ThreadingHTTPServer(("127.0.0.1", 0), backend.Handler)
    finally:
        socket.getfqdn = original_getfqdn
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        with urllib.request.urlopen(
            f"http://127.0.0.1:{server.server_port}/corpus/valid", timeout=2,
        ) as response:
            assert response.status == 200
            json.load(response)
        with backend.LOCK:
            events = list(backend.EVENTS)
        assert len(events) == 1
        assert events[0]["httpMethod"] == "GET"
        assert events[0]["path"] == "/corpus/valid"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


def check_partial_corpus_diagnostic():
    original = (
        verify.EVIDENCE, verify.CORPUS_REVISION, verify.CORPUS_PARTIAL_RECORDS,
        verify.CORPUS_CURRENT_RECORD, verify.evidence_snapshot,
    )
    with tempfile.TemporaryDirectory() as directory:
        current = {"fixture": "fault-injected", "actualAcceptance": None}
        verify.EVIDENCE = Path(directory)
        verify.CORPUS_REVISION = "candidate"
        verify.CORPUS_PARTIAL_RECORDS = [current]
        verify.CORPUS_CURRENT_RECORD = current
        verify.evidence_snapshot = lambda: {"backend-primary": []}
        verify.record_corpus_failure(RuntimeError("admin checker fault"))
        observed = json.loads((Path(directory) / "corpus-candidate.json").read_text())
        assert observed["fixtures"][0]["diagnosticError"] == "admin checker fault"
        assert observed["fixtures"][0]["backendEvents"] == {"backend-primary": []}
    (
        verify.EVIDENCE, verify.CORPUS_REVISION, verify.CORPUS_PARTIAL_RECORDS,
        verify.CORPUS_CURRENT_RECORD, verify.evidence_snapshot,
    ) = original


if __name__ == "__main__":
    check_transient_lds_publish_timeout()
    check_rejection_poll_classification()
    check_corpus_get_fixture()
    check_partial_corpus_diagnostic()
    print("runtime orchestration fault-injection self-tests passed")

"""
Zenith Python Parallel Engine
Flask HTTP server that polls mock TCP instruments via a ThreadPoolExecutor.
Mock instruments are started at server boot on ports 9001–9050 (shared with
the Go engine — Python polls Go's stable listeners directly).

Production usage:
    gunicorn --workers 1 --threads 64 --bind 0.0.0.0:8000 server:app
"""
from __future__ import annotations

import logging
import os
import socket
import threading
import time
import random
from concurrent.futures import ThreadPoolExecutor, as_completed

from flask import Flask, jsonify, request
from flask_cors import CORS

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Configuration (environment-variable driven)
# ---------------------------------------------------------------------------
INSTRUMENT_PORT_BASE: int = 9001          # Shared mock instruments: 9001–9050 (same as Go engine)
MAX_INSTRUMENTS: int = 50
ALLOWED_ORIGIN: str = os.environ.get("ALLOWED_ORIGIN", "http://localhost:3000")
PORT: int = int(os.environ.get("PORT", "8000"))

# ---------------------------------------------------------------------------
# App
# ---------------------------------------------------------------------------
app = Flask(__name__)
CORS(app, origins=[ALLOWED_ORIGIN])

# ---------------------------------------------------------------------------
# Mock TCP instrument
# ---------------------------------------------------------------------------
_instruments_started = False
_instruments_lock = threading.Lock()


def _handle_instrument_connection(conn: socket.socket, meas_response: str) -> None:
    """Serve one client connection, always returning the stable pre-generated reading."""
    with conn:
        while True:
            try:
                data = conn.recv(1024)
            except OSError:
                break
            if not data:
                break
            command = data.decode(errors="replace").strip()
            if command == "*IDN?":
                conn.sendall(b"ZENITH-MOCK-B2901A-V2.6\n")
            elif command == ":MEAS?":
                conn.sendall(f"{meas_response}\n".encode())
            else:
                conn.sendall(b"ERR:INVALID_SCPI_CMD\n")


def _run_mock_instrument(port: int) -> None:
    # Generate stable readings once at startup so every :MEAS? poll on this
    # port returns the same V/I.  Both Go and Python share the same ports
    # (9001-9050), so when Go's instruments are already bound Python's bind
    # will fail gracefully and Python polls Go's stable instruments instead.
    v = 0.8 + random.random() * (2.1 - 0.2)
    i = 0.01 + random.random() * (0.59 - 0.01)
    meas_response = f"V:{v:.4f},I:{i:.4f}"

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        s.bind(("localhost", port))
    except OSError as exc:
        logger.warning(
            "Port %d already bound (Go engine owns it) — Python will poll it directly. %s",
            port, exc,
        )
        return
    s.listen()
    while True:
        try:
            conn, _ = s.accept()
        except OSError:
            break
        threading.Thread(
            target=_handle_instrument_connection, args=(conn, meas_response), daemon=True
        ).start()


def ensure_instruments_started() -> None:
    """Start all mock instrument servers exactly once (thread-safe)."""
    global _instruments_started
    with _instruments_lock:
        if _instruments_started:
            return
        for i in range(MAX_INSTRUMENTS):
            threading.Thread(
                target=_run_mock_instrument,
                args=(INSTRUMENT_PORT_BASE + i,),
                daemon=True,
            ).start()
        time.sleep(0.15)  # give all listeners time to bind
        _instruments_started = True
        logger.info(
            "Mock instruments started: ports %d–%d",
            INSTRUMENT_PORT_BASE,
            INSTRUMENT_PORT_BASE + MAX_INSTRUMENTS - 1,
        )


# ---------------------------------------------------------------------------
# Parallel polling
# ---------------------------------------------------------------------------

def _poll_instrument(device_id: str, port: int) -> dict:
    """Poll a single instrument and return a result dict (never raises)."""
    start = time.perf_counter()
    try:
        with socket.create_connection(("localhost", port), timeout=2) as s:
            s.sendall(b":MEAS?\n")
            # makefile gives reliable line-oriented reading; recv(N) is not
            # guaranteed to return a full line in a single call.
            with s.makefile("r") as f:
                data = f.readline().strip()
            latency_ms = (time.perf_counter() - start) * 1000
            return {"device": device_id, "pyLatency": f"{latency_ms:.3f}", "pyData": data or "NO_DATA"}
    except Exception as exc:
        latency_ms = (time.perf_counter() - start) * 1000
        logger.warning("Poll failed for %s on port %d: %s", device_id, port, exc)
        return {"device": device_id, "pyLatency": f"{latency_ms:.3f}", "pyData": "ERROR"}


# ---------------------------------------------------------------------------
# Module-level thread pool — reused across all requests to avoid the overhead
# of creating and destroying threads on every benchmark call.
# ---------------------------------------------------------------------------
_executor = ThreadPoolExecutor(max_workers=MAX_INSTRUMENTS, thread_name_prefix="zenith-poll")

# Eagerly start instruments at module load time so they are ready before the first
# HTTP request arrives. In WSGI mode (gunicorn), __name__ != "__main__", so the
# if-block at the bottom would be skipped — this ensures startup always happens.
ensure_instruments_started()


# ---------------------------------------------------------------------------
# HTTP endpoints
# ---------------------------------------------------------------------------

@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"}), 200


@app.route("/benchmark", methods=["GET"])
def benchmark():
    try:
        count = int(request.args.get("count", 5))
    except (ValueError, TypeError):
        count = 5
    count = max(1, min(count, MAX_INSTRUMENTS))

    # Pre-fill every slot with a safe error record so no response is ever None,
    # even if a future raises an unexpected exception.
    ordered: list[dict] = [
        {"device": f"SMU-{i + 1}", "pyLatency": "0.000", "pyData": "ERROR"}
        for i in range(count)
    ]
    cycle_start = time.perf_counter()

    futures = {
        _executor.submit(_poll_instrument, f"SMU-{i + 1}", INSTRUMENT_PORT_BASE + i): i
        for i in range(count)
    }
    for future in as_completed(futures):
        try:
            ordered[futures[future]] = future.result()
        except Exception as exc:  # pragma: no cover — defensive catch
            logger.error("Unexpected future error for index %d: %s", futures[future], exc)

    cycle_ms = (time.perf_counter() - cycle_start) * 1000

    return jsonify({
        "cycleTimeMs": f"{cycle_ms:.3f}",
        "measurements": ordered,
    })


if __name__ == "__main__":
    ensure_instruments_started()
    logger.info("ZENITH PYTHON PARALLEL ENGINE")
    logger.info(
        "Mock Instruments: %d active (ports %d–%d)",
        MAX_INSTRUMENTS, INSTRUMENT_PORT_BASE, INSTRUMENT_PORT_BASE + MAX_INSTRUMENTS - 1,
    )
    logger.info("HTTP Server: http://localhost:%d/benchmark?count=N", PORT)
    logger.warning("Running Flask dev server — use Gunicorn for production.")
    app.run(host="0.0.0.0", port=PORT, debug=False)

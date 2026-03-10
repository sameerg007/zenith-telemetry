"""
Zenith Python Parallel Engine
Flask HTTP server that polls real mock TCP instruments in parallel threads.
Mock instruments are pre-started at server boot on ports 9101-9150.
"""
import socket
import threading
import time
import random

from flask import Flask, request, jsonify
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

INSTRUMENT_PORT_BASE = 9101   # Python mock instruments: 9101–9150
MAX_INSTRUMENTS = 50

_instruments_started = False
_instruments_lock = threading.Lock()


# ---------------------------------------------------------------------------
# Mock TCP instrument
# ---------------------------------------------------------------------------

def _handle_instrument_connection(conn: socket.socket) -> None:
    with conn:
        while True:
            data = conn.recv(1024)
            if not data:
                break
            command = data.decode().strip()
            if command == "*IDN?":
                conn.sendall(b"ZENITH-MOCK-B2901A-V2.6\n")
            elif command == ":MEAS?":
                v = 0.8 + random.random() * (1.2 - 0.8)       # 0.8 V – 1.2 V
                i = 0.01 + random.random() * (0.05 - 0.01)     # 10 mA – 50 mA
                conn.sendall(f"V:{v:.4f},I:{i:.4f}\n".encode())
            else:
                conn.sendall(b"ERR:INVALID_SCPI_CMD\n")


def _run_mock_instrument(port: int) -> None:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("localhost", port))
    s.listen()
    while True:
        conn, _ = s.accept()
        threading.Thread(
            target=_handle_instrument_connection, args=(conn,), daemon=True
        ).start()


def _ensure_instruments_started() -> None:
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
        time.sleep(0.1)  # give listeners time to bind
        _instruments_started = True


# ---------------------------------------------------------------------------
# Parallel polling
# ---------------------------------------------------------------------------

def _poll_instrument(device_id: str, port: int, results: list) -> None:
    start = time.time()
    try:
        with socket.create_connection(("localhost", port), timeout=2) as s:
            s.sendall(b":MEAS?\n")
            data = s.recv(1024).decode().strip()
            latency_ms = (time.time() - start) * 1000
            results.append({"device_id": device_id, "data": data, "latency": latency_ms})
    except Exception:
        results.append({"device_id": device_id, "data": "ERROR", "latency": 0.0})


# ---------------------------------------------------------------------------
# HTTP endpoint
# ---------------------------------------------------------------------------

@app.route("/benchmark", methods=["GET"])
def benchmark():
    _ensure_instruments_started()

    try:
        count = int(request.args.get("count", 5))
    except ValueError:
        count = 5
    count = max(1, min(count, MAX_INSTRUMENTS))

    cycle_start = time.time()
    threads: list[threading.Thread] = []
    results: list[dict] = []

    for i in range(count):
        device_id = f"SMU-{i + 1}"
        port = INSTRUMENT_PORT_BASE + i
        t = threading.Thread(target=_poll_instrument, args=(device_id, port, results))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    cycle_ms = (time.time() - cycle_start) * 1000

    return jsonify({
        "cycleTimeMs": f"{cycle_ms:.3f}",
        "measurements": [
            {
                "device": r["device_id"],
                "pyLatency": f"{r['latency']:.3f}",
                "pyData": r["data"],
            }
            for r in results
        ],
    })


if __name__ == "__main__":
    _ensure_instruments_started()
    print("--- ZENITH PYTHON PARALLEL ENGINE ---")
    print(f"Mock Instruments : {MAX_INSTRUMENTS} active "
          f"(ports {INSTRUMENT_PORT_BASE}–{INSTRUMENT_PORT_BASE + MAX_INSTRUMENTS - 1})")
    print("HTTP Server      : http://localhost:8000/benchmark?count=N")
    app.run(host="0.0.0.0", port=8000)

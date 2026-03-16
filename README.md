# Zenith Telemetry — Parallel Engine Benchmark

A full-stack real-time benchmark that proves **Go goroutines outperform Python threading** for concurrent I/O by polling identical mock SCPI instruments in parallel and comparing total cycle times side-by-side.

---

## How it works

```
Browser (Next.js :3000)
    │
    ├── GET /benchmark?count=N ──► Go Engine (:8080)
    │                               └── 50 goroutines poll mock TCP instruments (:9001-:9050)
    │
    └── GET /benchmark?count=N ──► Python Engine (:8000)
                                    └── ThreadPoolExecutor polls the same instruments (:9001-:9050)
```

Both engines hit the **same mock instruments** so the V/I data values are identical — only the parallel execution speed (cycle time) differs. The dashboard highlights which engine finished the full batch faster.

---

## Project structure

```
zenith-ui/
├── backend/
│   ├── go/                      # Go parallel engine (port 8080)
│   │   ├── main.go              # HTTP server, worker pool, graceful shutdown
│   │   ├── go.mod
│   │   └── pkg/
│   │       ├── engine/          # TCP instrument poller with I/O size cap
│   │       └── simulator/       # Mock SCPI TCP instruments (ports 9001-9050)
│   └── python/                  # Python parallel engine (port 8000)
│       ├── server.py            # Flask server, ThreadPoolExecutor, input validation
│       └── requirements.txt
└── src/
    └── app/
        └── benchmark/
            ├── page.tsx         # Live benchmark dashboard (AG Grid)
            ├── benchmarkApi.ts  # Typed fetch client with timeout
            └── config.ts        # Shared UI constants
```

---

## Prerequisites

| Tool | Minimum version | Check |
|---|---|---|
| Node.js | 18 | `node -v` |
| npm | 9 | `npm -v` |
| Go | 1.21 | `go version` |
| Python | 3.10 | `python3 --version` |

---

## Installation

### 1. Clone the repository

```bash
git clone <repo-url>
cd zenith-ui
```

### 2. Install frontend dependencies

```bash
npm install
```

### 3. Install Python dependencies

```bash
pip install -r backend/python/requirements.txt
```

> If you use conda or a virtual environment, activate it first:
> ```bash
> # conda
> conda activate base
> # or venv
> source .venv/bin/activate
> ```

### 4. Configure environment variables (optional)

The defaults work out of the box for local development. To override:

```bash
cp .env.local.example .env.local
# Edit .env.local if your backend ports differ
```

---

## Running the app

All three processes must run simultaneously. Open **three terminal tabs**.

### Terminal 1 — Go engine

```bash
cd backend/go
go run .
```

Expected output:
```
time=... level=INFO msg="ZENITH GO PARALLEL ENGINE" instruments=50 port_range=9001-9050 listen=http://localhost:8080
```

### Terminal 2 — Python engine

```bash
cd backend/python
python3 server.py
```

Expected output:
```
... [INFO] __main__: ZENITH PYTHON PARALLEL ENGINE
... [INFO] werkzeug: Running on http://127.0.0.1:8000
```

> **Note:** Python attempts to bind to ports 9001-9050 at startup. If Go is already running, those binds will fail with a warning — that is expected and correct. Python will then poll Go's instruments directly.

### Terminal 3 — Next.js frontend

```bash
npm run dev
```

Expected output:
```
▲ Next.js — Local: http://localhost:3000
```

### Open the dashboard

Navigate to **http://localhost:3000/benchmark** in your browser.

---

## Health checks

Once all three are running you can verify each service:

```bash
curl http://localhost:8080/health   # {"status":"ok"}
curl http://localhost:8000/health   # {"status":"ok"}
curl http://localhost:3000          # 200 / 307
```

---

## Production usage

### Go engine

```bash
cd backend/go
go build -o zenith-go .
PORT=8080 ALLOWED_ORIGIN=https://your-domain.com ./zenith-go
```

### Python engine

Use Gunicorn instead of the Flask dev server:

```bash
cd backend/python
gunicorn --workers 1 --threads 64 --bind 0.0.0.0:8000 server:app
```

### Next.js frontend

```bash
npm run build
npm start
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_GO_API_URL` | `http://127.0.0.1:8080` | Go engine base URL (browser-visible) |
| `NEXT_PUBLIC_PY_API_URL` | `http://127.0.0.1:8000` | Python engine base URL (browser-visible) |
| `PORT` | `8080` / `8000` | Listening port for each backend |
| `ALLOWED_ORIGIN` | `http://localhost:3000` | CORS allowed origin for each backend |

---

## Security measures

- **Input validation** — `count` query parameter returns `400` if missing, non-integer, or out of range (1-50).
- **Instrument response validation** — every TCP response is matched against `V:X.XXXX,I:X.XXXX` before it enters the API. Malformed data (including injection payloads) is replaced with `INVALID_RESPONSE`.
- **Response size cap** — Go reads at most 512 bytes per instrument connection (`io.LimitedReader`).
- **Connection idle timeout** — Mock instrument goroutines close stale connections after 30 s.
- **Worker pool** — Go caps concurrent goroutines at 100; Python uses a fixed-size `ThreadPoolExecutor`.
- **Request timeout** — Each benchmark request times out after 10 s server-side.
- **Client timeout** — The browser fetch client times out after 15 s.
- **HTTP security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Referrer-Policy`, `Permissions-Policy` applied to all Next.js responses.
- **CORS** — Both backends restrict cross-origin access to the configured `ALLOWED_ORIGIN`.
- **Slowloris mitigation** — Go HTTP server has strict `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.
# zenith-telemetry

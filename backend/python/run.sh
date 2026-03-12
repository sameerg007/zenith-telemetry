#!/bin/bash
# Starts the Zenith Python Parallel Engine (Gunicorn + mock TCP instruments)
set -euo pipefail

cd "$(dirname "$0")"

if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

source venv/bin/activate
pip install -q -r requirements.txt

echo "Starting Zenith Python Parallel Engine on port ${PORT:-8000}..."
exec venv/bin/gunicorn \
    --workers 1 \
    --threads 64 \
    --bind "0.0.0.0:${PORT:-8000}" \
    --timeout 30 \
    --access-logfile - \
    server:app

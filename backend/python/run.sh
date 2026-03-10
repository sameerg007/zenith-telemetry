#!/bin/bash
# Starts the Zenith Python Parallel Engine (Flask + mock TCP instruments)

cd "$(dirname "$0")"

if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

source venv/bin/activate
pip install -q flask flask-cors

echo "Starting Zenith Python Parallel Engine on port 8000..."
python server.py

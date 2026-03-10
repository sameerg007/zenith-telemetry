#!/bin/bash

# Navigate to the directory containing this script
cd "$(dirname "$0")"

# Create a virtual environment named 'venv' if it doesn't exist
if [ ! -d "venv" ]; then
    python3 -m venv venv
fi

# Activate the virtual environment and install dependencies
source venv/bin/activate
pip install flask flask-cors

# Run the Python server
python server.py
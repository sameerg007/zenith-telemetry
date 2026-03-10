#!/bin/bash
# Delegates to the Zenith Python Parallel Engine in backend/python/
exec "$(dirname "$0")/../../../backend/python/run.sh" "$@"
#!/bin/bash

# Script to run claude-squad with proper TTY allocation
# This can help when running in environments without direct TTY access

echo "Building Claude Squad..."
go build

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Starting Claude Squad..."
echo "Note: This requires a proper terminal environment with TTY support"

# Try to run with different TTY configurations
if command -v script &> /dev/null; then
    echo "Using 'script' command for TTY allocation..."
    script -q /dev/null ./claude-squad "$@"
elif command -v expect &> /dev/null; then
    echo "Using 'expect' for TTY allocation..."
    expect -c "spawn ./claude-squad $@; interact"
else
    echo "Running directly (may not work in all environments)..."
    ./claude-squad "$@"
fi
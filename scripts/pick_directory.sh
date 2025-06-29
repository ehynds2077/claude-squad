#!/bin/bash

# Shell wrapper for directory picker using nvim + Oil.nvim
# Usage: ./pick_directory.sh [output_file]

# Default output file
OUTPUT_FILE=${1:-"/tmp/claude_squad_selected_dir"}

# Path to the Lua script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LUA_SCRIPT="$SCRIPT_DIR/pick_directory.lua"

# Check if nvim is available
if ! command -v nvim &> /dev/null; then
    echo "Error: Neovim is not installed or not available in PATH"
    echo "" > "$OUTPUT_FILE"
    exit 1
fi

# Check if Oil.nvim is available (basic check)
if ! nvim --headless -c "lua require('oil')" -c "qa" 2>/dev/null; then
    echo "Error: Oil.nvim is not installed or not configured properly"
    echo "Please install Oil.nvim: https://github.com/stevearc/oil.nvim"
    echo "" > "$OUTPUT_FILE"
    exit 1
fi

# Clear the output file
echo "" > "$OUTPUT_FILE"

# Run nvim with the directory picker script
nvim -c "luafile $LUA_SCRIPT" "$OUTPUT_FILE"

# Check if a directory was selected
if [ -s "$OUTPUT_FILE" ]; then
    # File has content - directory was selected
    SELECTED_DIR=$(cat "$OUTPUT_FILE")
    echo "Selected directory: $SELECTED_DIR"
    exit 0
else
    # File is empty - selection was cancelled
    echo "Directory selection cancelled"
    exit 1
fi
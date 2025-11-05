#!/usr/bin/env bash

# Script to build plugins from a configuration file or command line arguments
# Usage:
#   ./build-plugin.sh <config-file> [output-file] [-o /path/output.so]
#   ./build-plugin.sh <repo@version> [repo@version...] [--output output-file] [-o /path/output.so]

set -e

# Default plugin output path
PLUGIN_OUTPUT="/dist/plugins.so"

# Parse arguments to extract -o flag for go build and collect remaining args for generate-compile.sh
GENERATE_ARGS=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o)
            shift
            PLUGIN_OUTPUT="$1"
            shift
            ;;
        *)
            GENERATE_ARGS+=("$1")
            shift
            ;;
    esac
done

./script/generate-compile.sh "${GENERATE_ARGS[@]}"

CGO_ENABLED=1 go build -o "$PLUGIN_OUTPUT" -buildmode=plugin ./cmd/plugindl

echo ""
echo "Plugin built: $PLUGIN_OUTPUT"

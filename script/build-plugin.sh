#!/usr/bin/env bash

# Script to build plugins from a configuration file or command line arguments
# Usage:
#   ./build-plugin.sh <config-file> [output-file] [-o /path/output.so]
#   ./build-plugin.sh <repo@version> [repo@version...] [--output output-file] [-o /path/output.so]

set -e

# Parse arguments to extract -o flag for go build and collect remaining args for generate-compile.sh
GENERATE_ARGS=()
CUSTOM_OUTPUT=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o)
            shift
            CUSTOM_OUTPUT="$1"
            shift
            ;;
        *)
            GENERATE_ARGS+=("$1")
            shift
            ;;
    esac
done

# Run generate-compile.sh and check if it succeeds
if ! ./script/generate-compile.sh "${GENERATE_ARGS[@]}"; then
    echo "Error: generate-compile.sh failed"
    exit 1
fi

# Determine plugin output path
if [ -n "$CUSTOM_OUTPUT" ]; then
    PLUGIN_OUTPUT="$CUSTOM_OUTPUT"
else
    # Check if we have exactly one non-file argument (single plugin mode)
    if [ ${#GENERATE_ARGS[@]} -eq 1 ] && [ ! -f "${GENERATE_ARGS[0]}" ]; then
        # Extract plugin name from repo path and sanitize it
        PLUGIN_NAME="${GENERATE_ARGS[0]}"
        # Replace / and @ with _
        PLUGIN_NAME="${PLUGIN_NAME//\//_}"
        PLUGIN_NAME="${PLUGIN_NAME//@/_}"

        # Get GOARCH and GOOS
        GOARCH=$(go env GOARCH)
        GOOS=$(go env GOOS)

        # Build dynamic output path
        PLUGIN_OUTPUT="/dist/${PLUGIN_NAME}-${GOOS}-${GOARCH}.so"
    else
        # Default for file mode or multiple plugins
        PLUGIN_OUTPUT="/dist/plugins.so"
    fi
fi

CGO_ENABLED=1 go build -o "$PLUGIN_OUTPUT" -buildmode=plugin ./cmd/plugindl

echo ""
echo "Plugin built: $PLUGIN_OUTPUT"

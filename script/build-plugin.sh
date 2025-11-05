#!/usr/bin/env bash

# Script to build plugins from a configuration file or command line arguments
# Usage:
#   ./build-plugin.sh <config-file> [output-file]
#   ./build-plugin.sh <repo@version> [repo@version...] [--output output-file]

./script/generate-compile.sh "$@"

CGO_ENABLED=1 go build -o /dist/plugins.so -buildmode=plugin ./cmd/plugindl

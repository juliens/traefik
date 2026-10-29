#!/usr/bin/env bash
# Copies the plugins module of Traefik, with its dependencies, into the plugins-local GOPATH given as argument:
# Traefik loads its providers and middlewares from there (experimental.localPlugins).

set -e -o pipefail

DEST="${1:?usage: plugins-local.sh <plugins-local directory>}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${DEST}/src/github.com/traefik/traefik/v3/plugins"

mkdir -p "${TARGET}"
TARGET="$(cd "${TARGET}" && pwd)"
rsync -a --delete --exclude vendor "${ROOT}/plugins/" "${TARGET}/"

cd "${ROOT}/plugins"
go mod vendor -o "${TARGET}/vendor"

#!/usr/bin/env bash

docker build -t builder --file buildplugin.Dockerfile .

docker build -t traefik/traefik:cgo --file cgo.Dockerfile .

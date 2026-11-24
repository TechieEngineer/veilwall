#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v go >/dev/null || { echo 'Install Go 1.23+ (a supported release recommended).'; exit 1; }
command -v npm >/dev/null || { echo 'Install Node.js 22.12+ and npm.'; exit 1; }
go mod tidy
if [[ -f web/package-lock.json ]]; then npm --prefix web ci; else npm --prefix web install; fi
npm --prefix web test
npm --prefix web run build
go vet ./...
go test ./...
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/veilwall ./cmd/veilwall
echo 'Ready: ./bin/veilwall --demo'

#!/bin/sh
# Собирает SvinoVoyna.exe для Windows (Go 1.24+; cgo и Python не нужны).
set -e
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H=windowsgui -s -w" -o dist/SvinoVoyna.exe ./cmd/svinovoyna
cp README.md dist/README.md
echo "Готово: dist/SvinoVoyna.exe"

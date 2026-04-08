#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_BIN="${GO_BIN:-go}"
DIST="$ROOT/dist"

rm -rf "$DIST"
mkdir -p "$DIST"

"$GO_BIN" version >/dev/null

windows_dir="$DIST/cc98-autosign-fast-windows-amd64"
mkdir -p "$windows_dir"
cp "$ROOT/.env.example" "$windows_dir/.env"
cp "$ROOT/README.md" "$windows_dir/README.md"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$windows_dir/cc98-autosign-fast.exe" ./src

linux_dir="$DIST/cc98-autosign-fast-linux-amd64"
mkdir -p "$linux_dir"
cp "$ROOT/.env.example" "$linux_dir/.env"
cp "$ROOT/README.md" "$linux_dir/README.md"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$linux_dir/cc98-autosign-fast" ./src

(
  cd "$DIST"
  zip -rq cc98-autosign-fast-windows-amd64.zip cc98-autosign-fast-windows-amd64
  tar -czf cc98-autosign-fast-linux-amd64.tar.gz cc98-autosign-fast-linux-amd64
)

echo "Release artifacts:"
echo " - $DIST/cc98-autosign-fast-windows-amd64.zip"
echo " - $DIST/cc98-autosign-fast-linux-amd64.tar.gz"

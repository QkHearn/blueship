#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DIST="$ROOT/dist"
mkdir -p "$DIST"
cd "$ROOT"

version="${VERSION:-dev}"
ldflags="-s -w -X main.version=${version}"

build() {
  local os="$1" arch="$2" out="$3"
  echo "building $out ..."
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$DIST/$out" .
}

build darwin arm64 blueship-darwin-arm64
build darwin amd64 blueship-darwin-amd64
build windows amd64 blueship-windows-amd64.exe
build linux amd64 blueship-linux-amd64
build linux arm64 blueship-linux-arm64

# local convenience symlink/copy
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) cp -f "$DIST/blueship-darwin-arm64" "$DIST/blueship" ;;
  Darwin-x86_64) cp -f "$DIST/blueship-darwin-amd64" "$DIST/blueship" ;;
  Linux-x86_64) cp -f "$DIST/blueship-linux-amd64" "$DIST/blueship" ;;
  Linux-aarch64) cp -f "$DIST/blueship-linux-arm64" "$DIST/blueship" ;;
esac

chmod +x "$DIST"/blueship* 2>/dev/null || true
ls -la "$DIST"
echo "done"

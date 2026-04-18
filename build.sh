#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
DIST_DIR="$ROOT_DIR/dist"
APP_NAME="gantt"
VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"

if ! command -v go >/dev/null 2>&1; then
  echo "go not found in PATH" >&2
  exit 1
fi

build_target() {
  local goos="$1"
  local goarch="$2"
  local ext="$3"
  local target_name="${APP_NAME}-${goos}-${goarch}"
  local target_dir="$DIST_DIR/$target_name"
  local binary_path="$target_dir/${APP_NAME}${ext}"
  local archive_path="$DIST_DIR/${target_name}-${VERSION}.zip"

  mkdir -p "$target_dir"

  echo "==> building $target_name"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="-s -w" -o "$binary_path" "$ROOT_DIR"

  cat > "$target_dir/README.txt" <<EOF
$APP_NAME

Run:
  macOS: ./$(basename "$binary_path")
  Windows: $(basename "$binary_path")

Default URL:
  http://localhost:8080

This binary has embedded templates and static assets.
EOF

  (
    cd "$target_dir"
    zip -qr "$archive_path" .
  )
}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "==> tidy check"
go test ./...

build_target darwin amd64 ""
build_target darwin arm64 ""
build_target windows amd64 ".exe"

if [[ "$(uname -s)" == "Darwin" ]] && command -v lipo >/dev/null 2>&1; then
  echo "==> creating universal macOS binary"
  UNIVERSAL_DIR="$DIST_DIR/${APP_NAME}-darwin-universal"
  mkdir -p "$UNIVERSAL_DIR"
  lipo -create \
    "$DIST_DIR/${APP_NAME}-darwin-amd64/${APP_NAME}" \
    "$DIST_DIR/${APP_NAME}-darwin-arm64/${APP_NAME}" \
    -output "$UNIVERSAL_DIR/${APP_NAME}"

  cat > "$UNIVERSAL_DIR/README.txt" <<EOF
$APP_NAME

Universal macOS binary (Intel + Apple Silicon).

Run:
  ./$(basename "$APP_NAME")

Default URL:
  http://localhost:8080
EOF

  (
    cd "$UNIVERSAL_DIR"
    zip -qr "$DIST_DIR/${APP_NAME}-darwin-universal-${VERSION}.zip" .
  )
fi

echo "==> done"
echo "Artifacts in: $DIST_DIR"
#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
mkdir -p "$DIST_DIR"

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
DATE="${DATE:-$(date -u +%Y-%m-%d)}"
LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE"

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "$platform"
  output="streamy-${GOOS}-${GOARCH}"
  if [[ "$GOOS" == "windows" ]]; then
    output+=".exe"
  fi
  echo "Building $output"
  GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "$DIST_DIR/$output" ./cmd/streamy

done

echo "Build artifacts in $DIST_DIR"

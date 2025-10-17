#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🔎 Checking for Go import cycles..."

PACKAGES=(
  "./cmd/..."
  "./internal/..."
  "./pkg/..."
  "./tests/..."
)

if ! output=$(go list "${PACKAGES[@]}" 2>&1); then
  echo "$output"
  if echo "$output" | grep -qi "import cycle"; then
    echo "❌ Import cycle detected. See output above for details."
  else
    echo "❌ go list failed while checking for cycles."
  fi
  exit 1
fi

echo "✅ No import cycles detected by go list."

echo "📈 Capturing module dependency graph snapshot..."
go mod graph > /tmp/go-mod-graph.txt
echo "Saved module graph to /tmp/go-mod-graph.txt for reference."


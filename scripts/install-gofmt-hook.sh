#!/bin/bash

# Install git pre-commit hook for gofmt
# This script sets up the pre-commit hook to ensure Go code formatting

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOKS_DIR="$(git rev-parse --git-dir)/hooks"
PRE_COMMIT_HOOK="$HOOKS_DIR/pre-commit"

if [ ! -d "$HOOKS_DIR" ]; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Copy the pre-commit hook
cp "$SCRIPT_DIR/githooks/pre-commit" "$PRE_COMMIT_HOOK"

# Make it executable
chmod +x "$PRE_COMMIT_HOOK"

echo "✅ Pre-commit hook installed successfully!"
echo ""
echo "The hook will run gofmt on all staged Go files before each commit."
echo "If any files need formatting, the commit will be blocked with instructions."
echo ""
echo "To test the hook, try making a formatting violation and committing."
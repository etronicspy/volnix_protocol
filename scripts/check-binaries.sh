#!/bin/bash
# Script to check and remove binaries from project root
# According to project rules, binaries must be in build/ directory

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FOUND_BINARIES=()

# Check for common binary names in root
for binary in volnixd volnixd-standalone volnixd.exe volnixd-standalone.exe; do
    if [ -f "$binary" ]; then
        FOUND_BINARIES+=("$binary")
    fi
done

if [ ${#FOUND_BINARIES[@]} -gt 0 ]; then
    echo "❌ ERROR: Binaries found in project root:" >&2
    for binary in "${FOUND_BINARIES[@]}"; do
        echo "   - $binary" >&2
    done
    echo "" >&2
    echo "According to project rules, binaries must be in build/ directory." >&2
    echo "Removing binaries from root..." >&2
    for binary in "${FOUND_BINARIES[@]}"; do
        rm -f "$binary"
        echo "   ✅ Removed: $binary" >&2
    done
    echo "" >&2
    echo "💡 Tip: Use 'make build' or 'make build-standalone' to build binaries correctly." >&2
    exit 1
else
    echo "✅ No binaries found in project root"
    exit 0
fi





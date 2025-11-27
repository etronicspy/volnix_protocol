#!/bin/bash
# Script to check and move binaries from project root to build/ directory
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
    echo "⚠️  WARNING: Binaries found in project root:" >&2
    for binary in "${FOUND_BINARIES[@]}"; do
        echo "   - $binary" >&2
    done
    echo "" >&2
    echo "According to project rules, binaries must be in build/ directory." >&2
    echo "Moving binaries to build/ directory..." >&2
    
    # Ensure build directory exists
    mkdir -p "$ROOT_DIR/build"
    
    for binary in "${FOUND_BINARIES[@]}"; do
        # Check if target already exists
        if [ -f "$ROOT_DIR/build/$binary" ]; then
            # If target exists, remove the one in root (it's likely outdated)
            rm -f "$binary"
            echo "   ✅ Removed outdated: $binary (newer version exists in build/)" >&2
        else
            # Move to build directory
            mv "$binary" "$ROOT_DIR/build/$binary"
            echo "   ✅ Moved: $binary -> build/$binary" >&2
        fi
    done
    echo "" >&2
    echo "💡 Tip: Use 'make build' or 'make build-standalone' to build binaries correctly." >&2
    echo "💡 Never run 'go build ./cmd/volnixd' without -o flag - it will create binary in root!" >&2
    exit 0  # Changed to 0 - it's a warning, not an error, since we fixed it
else
    echo "✅ No binaries found in project root"
    exit 0
fi








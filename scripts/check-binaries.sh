#!/usr/bin/env bash
# Check that binaries are not in project root (should be in build/)
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
for f in volnixd volnixd.exe volnixd-standalone volnixd-standalone.exe; do
  if [ -f "$f" ]; then
    echo "Warning: $f found in project root. Move to build/ or run: make clean"
    exit 1
  fi
done
exit 0

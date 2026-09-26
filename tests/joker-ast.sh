#!/usr/bin/env bash
# Passed to forked tests as their interpreter, so nested subprocesses also use AST.
set -e
root="$(cd "$(dirname "$0")/.." && pwd)"
exec "$root/joker" --no-vm "$@"

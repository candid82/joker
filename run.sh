#!/usr/bin/env bash

build() {
  go clean
  go generate ./...
  go vet ./...
  go build
}

set -e  # Exit on error.

build

if [ "$1" == "-v" ]; then
  ./joker -e '(print "\nLibraries available in this build:\n  ") (loaded-libs) (println)'
fi

SUM256="$(go run tools/sum256dir/main.go std)"
OUT="$(cd std; ../joker generate-std.joke 2>&1 | grep -v 'WARNING:.*already refers' | grep '.')" || : # grep returns non-zero if no lines match
if [ -n "$OUT" ]; then
    echo "$OUT"
    echo >&2 "Unable to generate fresh library files; exiting."
    exit 2
fi
NEW_SUM256="$(go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
  echo 'std has changed, rebuilding...'
  build
fi

./joker "$@"

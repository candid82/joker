#!/usr/bin/env bash

set -x

build() {
  go clean
  rm -f core/a_*.go  # In case switching from a gen-code branch or similar (any existing files might break the build here)
  go generate ./...
  (cd core; go fmt a_*.go > /dev/null)
  go vet -tags gen_code ./...
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
(cd std; go fmt ./... > /dev/null)
NEW_SUM256="$(go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
    echo 'std has changed, rebuilding...'
    build
fi

./joker "$@"

#!/usr/bin/env bash

[ -z "$KEEP_A_CODE_FILES" ] && KEEP_A_CODE_FILES=false
[ -z "$KEEP_A_DATA_FILES" ] && KEEP_A_DATA_FILES=false

build() {
  go clean
  $KEEP_A_CODE_FILES || rm -fv core/a_*code.go
  $KEEP_A_DATA_FILES || rm -fv core/a_*data.go
  go generate ./...
  [ -f OPTIMIZE-STARTUP.flag ] && (cd core; go run gen_code/gen_code.go && go fmt a_*.go > /dev/null)
  (cd core; go run gen_data/gen_data.go)
  go vet ./...
  go build
}

set -e  # Exit on error.

build

$KEEP_A_CODE_FILES && exit 0  # Going further than this does not yet work

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

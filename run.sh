#!/usr/bin/env bash

[ -z "$OPTIMIZE_STARTUP" ] && OPTIMIZE_STARTUP=$([ -f NO-OPTIMIZE-STARTUP.flag ] && echo false || echo true)
[ -z "$RUN_GEN_CODE" ] && RUN_GEN_CODE=$OPTIMIZE_STARTUP

build() {
  go clean
  rm -f core/a_*.go  # In case switching from a gen-code branch or similar (any existing files might break the build here)
  go generate ./...
  $RUN_GEN_CODE && (echo "Optimizing startup time..."; cd core; go run gen_code/gen_code.go && go fmt a_*.go > /dev/null)
  go vet ./...
  go build
  if $OPTIMIZE_STARTUP; then
      mv -f joker joker.slow
      go build -tags fast_init
      ln -f joker joker.fast
      echo "...built both joker.slow and (also named joker) joker.fast."
  else
      ln -f joker joker.slow
  fi
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

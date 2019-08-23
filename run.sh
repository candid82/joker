#!/usr/bin/env bash

build() {
  go clean
  GOOS="$GOHOSTOS" GOARCH="$GOHOSTARCH" go generate ./...
  go vet ./...
  go build
}

set -e  # Exit on error.

build

if [ "$1" == "-v" ]; then
  ./joker -e '(print "\nLibraries available in this build:\n  ") (loaded-libs) (println)'
fi

SUM256="$(GOOS="$GOHOSTOS" GOARCH="$GOHOSTARCH" go run tools/sum256dir/main.go std)"
(cd std; ../joker generate-std.joke 2> /dev/null)
NEW_SUM256="$(GOOS="$GOHOSTOS" GOARCH="$GOHOSTARCH" go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
  echo 'std has changed, rebuilding...'
  build
fi

./joker "$@"

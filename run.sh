#!/usr/bin/env bash

build() {
    go clean -x

    go generate ./...

    go tool vet -all -shadow=true ./

    go build
}

set -e  # Exit on error.

build

./joker -e '(print "\nLibraries available in this build:\n  ") *loaded-libs* (println)'

SUM256="$(go run tools/sum256dir/main.go std)"

(cd std; ../joker generate-std.joke)

NEW_SUM256="$(go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
    cat <<EOF
Rebuilding Joker, as the libraries have changed; then regenerating docs.

EOF
    build

    (cd docs; ../joker generate-docs.joke)
fi

./joker "$@"

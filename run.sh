#!/usr/bin/env bash

ex () {
    exit $?
}
trap ex ERR

build() {
    trap ex ERR

    go clean -x

    go generate ./...

    $GOVET

    if [ -n "$SHADOW" ]; then
        go vet -all "$SHADOW" ./... && echo "Shadowed-variables check complete."
    fi

    go build
}

if which shadow >/dev/null 2>/dev/null; then
    # Install via: go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
    SHADOW="-vettool=$(which shadow)"
    GOVET="go vet -all ./..."  # Must be done as a separate step
elif $(go tool vet nonexistent.go 2>&1 | grep -q -v unsupported); then
    SHADOW="-shadow=true"  # Older version of go that supports 'go tool vet' and thus '-shadow=true'
    GOVET=""  # One-step vetting works with 'go tool vet -all -shadow=true'
else
    GOVET="go vet -all ./..."
    echo "Not performing shadowed-variables check; consider installing shadow tool via:"
    echo "  go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow"
    echo "and rebuilding."
fi

build

./joker -e '(print "\nLibraries available in this build:\n  ") (loaded-libs) (println)'

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

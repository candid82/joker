#!/usr/bin/env bash

go install

LIBS="$GOPATH/share/joker"

mkdir -p "$LIBS"

cd share

for f in *; do
    rm -fr "$LIBS/$f"
    cp -a "$f" "$LIBS/$f"
done

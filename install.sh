#!/usr/bin/env bash

go install

LIBS="$GOPATH/share/joker"

mkdir -p "$LIBS"

cp -a share/joker "$GOPATH/share/joker"

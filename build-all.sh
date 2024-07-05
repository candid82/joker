#!/usr/bin/env bash

export GOARCH=amd64
export GOOS=

for GOOS in darwin linux windows freebsd netbsd openbsd; do
    go build

    case "$GOOS" in
    darwin) os=mac ;;
    windows) os=win ;;
    *) os=$GOOS ;;
    esac
    [[ $GOOS = windows ]] && executable=joker.exe || executable=joker

    zip -9 joker-"$os"-"$GOARCH".zip "$executable"
done

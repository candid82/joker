#!/usr/bin/env bash
set -eu

export GOARCH=amd64
export GOOS=

for GOOS in darwin linux windows freebsd netbsd openbsd; do
    if ! go build; then
        echo \
            error: \
            "building for GOARCH=$GOARCH GOOS=$GOOS failed" \
            >/dev/stderr \
            ;
        continue
    fi

    case "$GOOS" in
    darwin) os=mac ;;
    windows) os=win ;;
    *) os=$GOOS ;;
    esac
    [[ $GOOS = windows ]] && executable=joker.exe || executable=joker

    zip -9 joker-"$os"-"$GOARCH".zip "$executable"
done

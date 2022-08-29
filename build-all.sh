#!/usr/bin/env bash

version=$1

if [ -z "$version" ]; then
  echo "Usage: build-all.sh <version>"
  exit
fi

JOKER_STD_OS=darwin ./run.sh --build-only
GOOS=darwin GOARCH=amd64 go build
zip joker-$version-mac-amd64.zip joker

JOKER_STD_OS=linux ./run.sh --build-only
GOOS=linux GOARCH=amd64 go build
zip joker-$version-linux-amd64.zip joker

JOKER_STD_OS=windows ./run.sh --build-only
GOOS=windows GOARCH=amd64 go build
zip joker-$version-win-amd64.zip joker.exe

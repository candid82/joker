#!/usr/bin/env bash

version=$1

if [ -z "$version" ]; then
  echo "Usage: build-all.sh <version>"
  exit
fi

GOOS=darwin GOARCH=amd64 go build
zip joker-$version-mac-amd64.zip joker
GOOS=linux GOARCH=amd64 go build
zip joker-$version-linux-amd64.zip joker
GOOS=windows GOARCH=amd64 go build
zip joker-$version-win-amd64.zip joker.exe

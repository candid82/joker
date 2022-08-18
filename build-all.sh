#!/usr/bin/env bash

version=$1

GOOS=darwin GOARCH=amd64 go build
zip joker-mac-amd64.zip joker
GOOS=linux GOARCH=amd64 go build
zip joker-linux-amd64.zip joker
GOOS=windows GOARCH=amd64 go build
zip joker-win-amd64.zip joker.exe

#!/usr/bin/env bash

GOOS=linux GOARCH=386 go generate ./...
GOOS=linux GOARCH=arm go build

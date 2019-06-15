#!/usr/bin/env bash

go generate ./... && go vet ./... && go build

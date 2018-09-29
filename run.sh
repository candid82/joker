#!/usr/bin/env bash

go generate ./... && go tool vet -shift=false ./ && go build && ./joker "$@"

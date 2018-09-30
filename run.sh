#!/usr/bin/env bash

go generate ./... && go tool vet -shift=false -shadow=true ./ && go build && ./joker "$@"

#!/usr/bin/env bash

go generate ./... && go tool vet ./ && go build && ./gclojure $@

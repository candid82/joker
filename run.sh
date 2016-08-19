#!/usr/bin/env bash

# go-bindata -o core/bindata.go data
go generate && go tool vet ./ && go build && ./gclojure $@

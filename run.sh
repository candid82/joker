#!/usr/bin/env bash

go-bindata data
go generate && go build && ./gclojure $@

#!/usr/bin/env bash

go-bindata data
go build && ./gclojure $@

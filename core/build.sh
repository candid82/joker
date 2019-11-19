#!/bin/bash

rm -fv a_*_{code,data}.go
go run gen_code/gen_code.go || exit $?

go fmt a_*.go
cd ..; go build

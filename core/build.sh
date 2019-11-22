#!/bin/bash

set -e  # Exit on error.
set -x  # Echo commands

rm -fv a_*_{code,data}.go a_code.go

# Build gen_data before generating code that would otherwise be
# unnecessarily compiled into it.
time go build -o gen_data/gen_data gen_data/gen_data.go

time go run gen_code/gen_code.go

time ./gen_data/gen_data

time go fmt a_*.go

cd ..; time go build

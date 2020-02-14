#!/bin/bash

# This is purely a helper script for developing the fast-startup version of Joker. It is not intended to be incorporated into official Joker.

set -e  # Exit on error.

if $(which gln >/dev/null 2>&1)
then
    LN=gln
else
    LN=ln
fi

if $(ls a_*_code.go > /dev/null 2>&1)
then
    NOW="_test_AA/$(date +%Y%m%d%H%M%S).dir"
    mkdir -p "$NOW"
    mv -iv a_*.go "$NOW" || :
    cp -aiv code.go gen_go/gen_go.go gen_code/gen_code.go "$NOW"
    [ -x ../joker ] && cp -aiv ../joker "$NOW"
    (git log -n 1; git status) > "$NOW/git.txt"
    $LN -sfTv "$(basename $NOW)" _test_AA/LATEST
else
    rm -fv a_*_data.go
fi

time=$(which time)

set -x  # Echo commands

$time go run -tags gen_data gen_data/gen_data.go --verbose

$time go run -tags gen_code gen_code/gen_code.go --verbose

$time go fmt a_*.go

JUSTVET=false

$JUSTVET && (cd ..; $time go vet ./...)

$JUSTVET || (cd ..; RUN_GEN_CODE=false KEEP_A_CODE_FILES=true KEEP_A_DATA_FILES=true $time ./run.sh -e '(loaded-libs)')

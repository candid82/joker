#!/bin/bash

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
    cp -aiv code.go gen_code/gen_code.go "$NOW"
    [ -x ../joker ] && cp -aiv ../joker "$NOW"
    (git log -n 1; git status) > "$NOW/git.txt"
    $LN -sfTv "$(basename $NOW)" _test_AA/LATEST
else
    rm -fv a_*_data.go
fi

time=$(which time)

set -x  # Echo commands

$time go run gen_data/gen_data.go --verbose

$time go run gen_code/gen_code.go --verbose

$time go fmt a_*.go

JUSTVET=true

$JUSTVET && (cd ..; $time go vet ./...)

$JUSTVET || (cd ..; KEEP_A_CODE_FILES=true KEEP_A_DATA_FILES=true BUILD_ARGS="-tags fast_init" $time ./run.sh -e '(loaded-libs)')

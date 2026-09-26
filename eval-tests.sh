#!/usr/bin/env bash

if [[ "$1" == "--no-vm" ]]; then
  exec ./tests/joker-ast.sh tests/run-eval-tests.joke "$@"
fi
exec ./joker tests/run-eval-tests.joke "$@"

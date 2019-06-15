#!/usr/bin/env bash

# Let test suite know to try.
[[ -t 0 ]] && export TTY_TESTS=1

./joker tests/run-eval-tests.joke "$@"

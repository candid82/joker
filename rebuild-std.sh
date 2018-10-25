#!/usr/bin/env bash

# Does part of what release.joke does.

(cd std; ../joker generate-std.joke)

(cd docs; ../joker generate-docs.joke)

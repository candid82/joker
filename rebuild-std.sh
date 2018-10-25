#!/usr/bin/env bash

# Does part of what release.joke does.

# When adding a new std namespace (e.g. joker.X.Y):
#
#   -  Add Joker wrapper to `std/X/Y.joke`.
#
#   -  Add Go implementation to `std/X/Y/Y_native.go`.
#
#   -  Edit `std/generate-std.joke` to add it to `namespaces`.
#
#   -  Add it to list of imports near to of `main.go`.
#
#   -  Add it to `*loaded-libs*` in `data/core.joke`.
#
#   -  Run this script.
#
#   -  `git add` newly generated files.

(cd std; ../joker generate-std.joke)

(cd docs; ../joker generate-docs.joke)

echo ""
echo "Remember to 'git add' newly generated files:"
echo ""
git status

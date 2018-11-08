#!/usr/bin/env bash

# Does part of what release.joke does.

# When adding a new std namespace (e.g. joker.X.Y):
#
#   -  Add Joker wrapper to `std/X/Y.joke`.
#
#   -  Add Go implementation to `std/X/Y/Y_native.go`.
#
#   -  Add it to list of imports near to of `main.go`.
#
#   -  Add it to `*loaded-libs*` in `core/data/core.joke`.
#
#   -  Build Joker (./run.sh -e '*loaded-libs*')
#
#   -  Edit `std/generate-std.joke` to add .X.Y to `namespaces`.
#
#   -  Run this script.
#
#   -  `git add` newly generated files.

(cd std; ../joker generate-std.joke)

echo ""
echo "Remember to 'git add' newly generated files:"
echo ""
git status

echo ""
echo "After rebuilding Joker, consider doing this and reviewing the resulting docs:"
echo ""
echo "  (cd docs; ../joker generate-docs.joke)"
echo ""

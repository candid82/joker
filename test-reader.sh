#!/usr/bin/env bash

filelist=$(find $1 -type f -name "*.clj")

for f in $filelist
do
  ERROR=$(./gclojure --parse $f 2>&1)
  if [ -n "$ERROR" ]; then
    echo $f
    echo $ERROR
  fi
done

#!/usr/bin/env bash

filelist=$(find $1 -type f -name "$2")

for f in $filelist
do
  ERROR=$(./joker --lint$3 $f 2>&1)
  if [ -n "$ERROR" ]; then
    echo "$ERROR"
  fi
done

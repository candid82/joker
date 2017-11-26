#!/usr/bin/env bash

filelist=$(find $1 -type f -name "$2" -not -name "project.clj" -not -name "user.clj")

for f in $filelist
do
  ERROR=$(./joker --lint $f 2>&1)
  if [ -n "$ERROR" ]; then
    echo $f
    echo "$ERROR"
  fi
done

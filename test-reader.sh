#!/usr/bin/env bash

filelist=$(find $1 -mindepth 1 -maxdepth 1 -type f)

for f in $filelist
do
  echo $f
  ./gclojure $f
done
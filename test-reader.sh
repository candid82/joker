#!/usr/bin/env bash

filelist=$(find $1 -type f -name "*.clj")

for f in $filelist
do
  echo $f
  ./gclojure --read $f
done

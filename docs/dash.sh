#!/usr/bin/env bash

rm -f ./joker.xml
../joker generate-xml.joke

pushd dash
  rm -f ./*.html
  rm -f ./*.css
  rm -rf joker.docset
	cp ../*.html ./
	cp ../*.css ./
	dashing build joker
	tar --exclude='.DS_Store' -cvzf ../joker.tgz joker.docset
popd

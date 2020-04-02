#!/usr/bin/env bash

rm -f ./joker.xml
../joker generate-xml.joke

pushd dash
  rm -f ./*.html
  rm -f ./*.css
  rm -f ./*.js
  rm -rf joker.docset
	cp ../*.html ./
	cp ../*.css ./
	cp ../*.js ./
	dashing build joker
	tar --exclude='.DS_Store' -cvzf ../joker.tgz joker.docset
popd

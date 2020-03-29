#!/usr/bin/env bash

pushd dash
  rm -f ./*.html
  rm -f ./*.css
  rm -f ./joker.xml
  rm -rf joker.docset
	cp ../*.html ./
	cp ../*.css ./
        ../joker generate-xml.joke
	dashing build joker
	tar --exclude='.DS_Store' -cvzf ../joker.tgz joker.docset
popd

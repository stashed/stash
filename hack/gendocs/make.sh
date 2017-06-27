#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscode/stash/hack/gendocs
go run main.go

cd $GOPATH/src/github.com/appscode/stash/docs/reference
sed -i 's/######\ Auto\ generated\ by.*//g' *
popd

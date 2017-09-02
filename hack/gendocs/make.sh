#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscode/stash/hack/gendocs
go run main.go
popd

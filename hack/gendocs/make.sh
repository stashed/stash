#!/usr/bin/env bash

pushd $GOPATH/src/stash.appscode.dev/stash/hack/gendocs
go run main.go
popd

#!/bin/bash

set -x -e

mkdir -p $GOPATH/src/github.com/appscode
cp -r stash $GOPATH/src/github.com/appscode
pushd $GOPATH/src/github.com/appscode/stash

./hack/builddeps.sh
./hack/make.py
./hack/make.py test unit

popd

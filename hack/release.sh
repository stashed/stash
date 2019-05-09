#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/stash.appscode.dev/stash"

export APPSCODE_ENV=prod

pushd $REPO_ROOT

rm -rf dist

./hack/docker/setup.sh
./hack/docker/setup.sh release

rm dist/.tag

popd

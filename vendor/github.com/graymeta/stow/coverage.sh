#!/usr/bin/env bash
set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/graymeta/stow"

pushd $REPO_ROOT

echo "" > coverage.txt

for d in $(go list ./... | grep -v -e vendor); do
    go test -v -race -coverprofile=profile.out -covermode=atomic "$d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

popd

#!/bin/bash
set -x

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/stash"

pushd $REPO_ROOT

export STASH_NAMESPACE=stash-dev
export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)

kubectl create ns $STASH_NAMESPACE
cat $REPO_ROOT/hack/dev/apiregistration.yaml | envsubst | kubectl apply -f -

popd

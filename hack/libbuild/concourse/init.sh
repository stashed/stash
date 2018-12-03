#!/usr/bin/env bash

BASE_DIR=$(pwd)
GOPATH=$(go env GOPATH)
REPO_ROOT=${REPO_ROOT:-"$GOPATH/src/github.com/$ORG_NAME/$REPO_NAME"}
PHARMER_VERSION="0.1.0-rc.5"
ONESSL_VERSION="0.8.0"
ClusterProvider=$ClusterProvider

# copy $REPO_ROOT to $GOPATH
mkdir -p $REPO_ROOT
cp -r $REPO_NAME/. $REPO_ROOT

# install all the dependencies and prepeare cluster
source "$REPO_ROOT/hack/libbuild/concourse/dependencies.sh"
source "$REPO_ROOT/hack/libbuild/concourse/cluster.sh"

pushd $REPO_ROOT

# changed name of branch
# this is necessary because operator image tag is based on branch name
# for parallel tests, if two test build image of same tag, it'll create problem
# one test may finish early and delete image while other is using it
git checkout -b $(git rev-parse --short HEAD)-$ClusterProvider
popd

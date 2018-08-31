#!/usr/bin/env bash

set -eoux pipefail

ORG_NAME=appscode
REPO_NAME=stash
OPERATOR_NAME=stash
APP_LABEL=stash #required for `kubectl describe deploy -n kube-system -l app=$APP_LABEL`

export APPSCODE_ENV=dev
export DOCKER_REGISTRY=appscodeci

# get concourse-common
pushd $REPO_NAME
git status # required, otherwise you'll get error `Working tree has modifications.  Cannot add.`. why?
git subtree pull --prefix hack/libbuild https://github.com/appscodelabs/libbuild.git master --squash -m 'concourse'
popd

source $REPO_NAME/hack/libbuild/concourse/init.sh

cp creds/gcs.json /gcs.json
cp creds/stash/.env $GOPATH/src/github.com/$ORG_NAME/$REPO_NAME/hack/config/.env

pushd $GOPATH/src/github.com/$ORG_NAME/$REPO_NAME

# install dependencies
./hack/builddeps.sh
./hack/docker/setup.sh build
./hack/docker/setup.sh push

./hack/deploy/stash.sh --docker-registry=$DOCKER_REGISTRY
./hack/make.py test e2e --v=3 --rbac=true --webhook=true --kubeconfig=/root/.kube/config --selfhosted-operator=true
popd

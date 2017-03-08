#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
REPO_ROOT=$GOPATH/src/github.com/appscode/restik

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=restik
RESTIC_VER=0.4.0

DIST=$REPO_ROOT/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
	export $(cat $DIST/.tag | xargs)
fi

clean() {
    pushd $REPO_ROOT/hack/docker
    rm -rf restic restik Dockerfile
    popd
}

build_binary() {
    pushd $REPO_ROOT
    ./hack/builddeps.sh
    ./hack/make.py build restik
    detect_tag $DIST/.tag
    popd
}

build_docker() {
    pushd $REPO_ROOT/hack/docker

    # Download restic
    cp $DIST/restik/restik-linux-amd64 restik
    chmod 755 restik

    # Download restic
    wget https://github.com/restic/restic/releases/download/v${RESTIC_VER}/restic_${RESTIC_VER}_linux_amd64.bz2
    bzip2 -d restic_${RESTIC_VER}_linux_amd64.bz2
    mv restic_${RESTIC_VER}_linux_amd64 restic
    chmod +x restic

    cat >Dockerfile <<EOL
FROM alpine

COPY restic /restic
COPY restik /restik

USER nobody:nobody
ENTRYPOINT ["/restik"]
EOL
    local cmd="docker build -t appscode/$IMG:$TAG ."
    echo $cmd; $cmd

    rm restik Dockerfile
    popd
}

build() {
    build_binary
    build_docker
}

docker_push() {
    if [ "$APPSCODE_ENV" = "prod" ]; then
        echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
        exit 1
    fi
    if [ "$TAG_STRATEGY" = "git_tag" ]; then
        echo "Are you trying to 'release' binaries to prod?"
        exit 1
    fi
    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

docker_release() {
    if [ "$APPSCODE_ENV" != "prod" ]; then
        echo "'release' only works in PROD env."
        exit 1
    fi
    if [ "$TAG_STRATEGY" != "git_tag" ]; then
        echo "'apply_tag' to release binaries and/or docker images."
        exit 1
    fi

    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

source_repo $@

#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
REPO_ROOT=$GOPATH/src/github.com/appscode/stash

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=stash
RESTIC_VER=${RESTIC_VER:-0.8.3}
NEW_RESTIC_VER=${NEW_RESTIC_VER:-0.9.4} # also update in restic wrapper library
RESTIC_BRANCH=${RESTIC_BRANCH:-stash-0.4.2}

DIST=$REPO_ROOT/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
  export $(cat $DIST/.tag | xargs)
fi

clean() {
  pushd $REPO_ROOT/hack/docker
  rm -rf restic stash Dockerfile
  popd
}

build_binary() {
  pushd $REPO_ROOT
  ./hack/builddeps.sh
  ./hack/make.py build stash
  detect_tag $DIST/.tag

  if [ $RESTIC_VER = 'SOURCE' ]; then
    rm -rf $DIST/restic
    cd $DIST
    clone https://github.com/appscode/restic.git
    cd restic
    checkout $RESTIC_BRANCH
    echo "Build binary using golang docker image"
    docker run --rm -ti \
      -v $(pwd):/go/src/github.com/restic/restic \
      -w /go/src/github.com/restic/restic golang:1.8.3-alpine go build ./cmd/restic
    mv restic $DIST/restic-bin
    rm -rf *
    mv $DIST/restic-bin $DIST/restic/restic
  else
    # Download restic
    rm -rf $DIST/restic
    mkdir $DIST/restic
    cd $DIST/restic
    # install restic 0.8.3
    wget https://github.com/restic/restic/releases/download/v${RESTIC_VER}/restic_${RESTIC_VER}_linux_amd64.bz2
    bzip2 -d restic_${RESTIC_VER}_linux_amd64.bz2
    mv restic_${RESTIC_VER}_linux_amd64 restic
    # install new restic
    wget https://github.com/restic/restic/releases/download/v${NEW_RESTIC_VER}/restic_${NEW_RESTIC_VER}_linux_amd64.bz2
    bzip2 -d restic_${NEW_RESTIC_VER}_linux_amd64.bz2
    mv restic_${NEW_RESTIC_VER}_linux_amd64 restic_${NEW_RESTIC_VER}
  fi

  popd
}

build_docker() {
  pushd $REPO_ROOT/hack/docker

  # Download restic
  cp $DIST/stash/stash-alpine-amd64 stash
  chmod 755 stash

  cp $DIST/restic/restic restic
  cp $DIST/restic/restic_${NEW_RESTIC_VER} restic_${NEW_RESTIC_VER}
  chmod 755 restic
  chmod 755 restic_${NEW_RESTIC_VER}

  cat >Dockerfile <<EOL
FROM alpine:3.8

RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY restic /bin/restic
COPY restic_${NEW_RESTIC_VER} /bin/restic_${NEW_RESTIC_VER}
COPY stash /bin/stash

ENTRYPOINT ["/bin/stash"]
EXPOSE 56789
EOL
  local cmd="docker build --pull -t $DOCKER_REGISTRY/$IMG:$TAG ."
  echo $cmd
  $cmd

  rm stash Dockerfile restic restic_${NEW_RESTIC_VER}
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
  hub_canary
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
  hub_up
}

source_repo $@

#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
REPO_ROOT=$GOPATH/src/stash.appscode.dev/stash

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=stash
NEW_RESTIC_VER=${NEW_RESTIC_VER:-0.9.5} # also update in restic wrapper library
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

  # Download restic
  rm -rf $DIST/restic
  mkdir $DIST/restic
  cd $DIST/restic
  # install new restic
  wget https://github.com/restic/restic/releases/download/v${NEW_RESTIC_VER}/restic_${NEW_RESTIC_VER}_linux_amd64.bz2
  bzip2 -d restic_${NEW_RESTIC_VER}_linux_amd64.bz2
  mv restic_${NEW_RESTIC_VER}_linux_amd64 restic_${NEW_RESTIC_VER}

  popd
}

build_docker() {
  pushd $REPO_ROOT/hack/docker

  # Download restic
  cp $DIST/stash/stash-alpine-amd64 stash
  chmod 755 stash

  cp $DIST/restic/restic_${NEW_RESTIC_VER} restic_${NEW_RESTIC_VER}
  chmod 755 restic_${NEW_RESTIC_VER}

  cat >Dockerfile <<EOL
FROM mysql:8.0.14

# add our user and group first to make sure their IDs get assigned consistently, regardless of whatever dependencies get added
RUN groupadd -r stash --gid=1005 \
    && useradd -r -g stash  --uid=1005 stash

RUN set -x \
    && apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY restic_${NEW_RESTIC_VER} /bin/restic_${NEW_RESTIC_VER}
COPY stash /bin/stash

USER stash

ENTRYPOINT ["/bin/stash"]
EXPOSE 56789
EOL
  local cmd="docker build --pull -t $DOCKER_REGISTRY/$IMG:$TAG ."
  echo $cmd
  $cmd

  rm stash Dockerfile restic_${NEW_RESTIC_VER}
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

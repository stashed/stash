#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/restik

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=restic
RESTIC_VER=0.4.0

build() {
	pushd $REPO_ROOT/hack/docker/restic

	# Download restic
	wget https://github.com/restic/restic/releases/download/v${RESTIC_VER}/restic_${RESTIC_VER}_linux_amd64.bz2
	bzip2 -d restic_${RESTIC_VER}_linux_amd64.bz2
	mv restic_${RESTIC_VER}_linux_amd64 restic

	# Download restik
	wget -O restik https://cdn.appscode.com/binaries/restik/$TAG/restik-linux-amd64
	chmod +x restik
	local cmd="docker build -t appscode/$IMG:${RESTIC_VER} ."
	echo $cmd; $cmd
	rm restik
	popd
}

binary_repo $@

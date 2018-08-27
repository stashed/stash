#!/usr/bin/env bash

set -x

# uninstall operator
./hack/deploy/stash.sh --uninstall --purge

# remove creds
rm -rf /gcs.json
rm -rf hack/config/.env

# remove docker images
source "hack/libbuild/common/lib.sh"
detect_tag ''

# delete docker image on exit
./hack/libbuild/docker.py del_tag $DOCKER_REGISTRY stash $TAG

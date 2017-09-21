#!/bin/bash

set -x
set -eou pipefail

./hack/docker/setup.sh
env APPSCODE_ENV=prod ./hack/docker/setup.sh release

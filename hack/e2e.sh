#!/usr/bin/env bash

# Copyright The Stash Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eou pipefail

export CGO_ENABLED=1
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

GINKGO_ARGS=${GINKGO_ARGS:-}
TEST_ARGS=${TEST_ARGS:-}
DOCKER_REGISTRY=${DOCKER_REGISTRY:-}

echo "Running e2e tests:"
cmd="ginkgo -r --v -race --progress --trace --noisyPendings=false ${GINKGO_ARGS} test -- --docker-registry=${DOCKER_REGISTRY} ${TEST_ARGS}"
echo $cmd
$cmd

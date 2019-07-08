#!/usr/bin/env bash

# Copyright 2019 AppsCode Inc.
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

export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

echo "Running e2e tests:"
ginkgo -r --v --progress --trace test -- \
    --docker-registry=${DOCKER_REGISTRY} \
    --image-tag=${TAG}                   \
    --storageclass=${STORAGE_CLASS}
echo

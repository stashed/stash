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

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/stash.appscode.dev/stash"

pushd $REPO_ROOT

echo "" >coverage.txt

for d in $(go list ./... | grep -v -e vendor -e test); do
  go test -v -race -coverprofile=profile.out -covermode=atomic "$d"
  if [ -f profile.out ]; then
    cat profile.out >>coverage.txt
    rm profile.out
  fi
done

popd

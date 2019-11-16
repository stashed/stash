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

export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

TARGETS="$@"

if [ -n "$TARGETS" ]; then
    echo "Running reimport.py"
    cmd="reimport3.py ${REPO_PKG} ${TARGETS}"
    $cmd
    echo

    echo "Running goimports:"
    cmd="goimports -w ${TARGETS}"
    echo "$cmd"
    $cmd
    echo

    echo "Running gofmt:"
    cmd="gofmt -s -w ${TARGETS}"
    echo "$cmd"
    $cmd
    echo
fi

echo "Running shfmt:"
cmd="find . -path ./vendor -prune -o -name '*.sh' -exec shfmt -l -w -ci -i 4 {} \;"
echo "$cmd"
eval "$cmd" # xref: https://stackoverflow.com/a/5615748/244009
echo

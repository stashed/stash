#!/bin/bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eou pipefail

# ref: https://gist.github.com/joshisa/297b0bc1ec0dcdda0d1625029711fa24
parse_url() {
    proto="$(echo $1 | grep :// | sed -e's,^\(.*://\).*,\1,g')"
    # remove the protocol
    url="$(echo ${1/$proto/})"

    IFS='/'                  # / is set as delimiter
    read -ra PARTS <<<"$url" # str is read into an array as tokens separated by IFS
    if [ ${PARTS[0]} != 'github.com' ] || [ ${#PARTS[@]} -ne 5 ]; then
        echo "failed to parse relase-tracker: $url"
        exit 1
    fi
    export RELEASE_TRACKER_OWNER=${PARTS[1]}
    export RELEASE_TRACKER_REPO=${PARTS[2]}
    export RELEASE_TRACKER_PR=${PARTS[4]}
}

RELEASE_TRACKER=${RELEASE_TRACKER:-}
GITHUB_BASE_REF=${GITHUB_BASE_REF:-}

while IFS=$': \r\t' read -r -u9 marker v; do
    case $marker in
        Release-tracker)
            export RELEASE_TRACKER=$(echo $v | tr -d '\r\t')
            ;;
        Release)
            export RELEASE=$(echo $v | tr -d '\r\t')
            ;;
    esac
done 9< <(git show -s --format=%b)

[ ! -z "$RELEASE_TRACKER" ] || {
    echo "Release-tracker url not found."
    exit 0
}

[ ! -z "$GITHUB_BASE_REF" ] || {
    echo "GitHub base ref not found."
    exit 0
}

parse_url $RELEASE_TRACKER
api_url="repos/${RELEASE_TRACKER_OWNER}/${RELEASE_TRACKER_REPO}/issues/${RELEASE_TRACKER_PR}/comments"

case $GITHUB_BASE_REF in
    master)
        msg="/ready-to-tag github.com/${GITHUB_REPOSITORY} ${GITHUB_SHA}"
        ;;
    *)
        msg="/cherry-picked github.com/${GITHUB_REPOSITORY} ${GITHUB_BASE_REF} ${GITHUB_SHA}"
        ;;
esac

hub api "$api_url" -f body="$msg"

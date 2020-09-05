#!/bin/bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the AppsCode Community License 1.0.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -x

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/stash.appscode.dev/stash"

pushd $REPO_ROOT

OS=""
ARCH=""
DOWNLOAD_URL=""
DOWNLOAD_DIR=""
TEMP_DIRS=()
ONESSL=""

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup() {
    rm -rf ca.crt ca.key server.crt server.key
    # remove temporary directories
    for dir in "${TEMP_DIRS[@]}"; do
        rm -rf "${dir}"
    done
}

# detect operating system
# ref: https://raw.githubusercontent.com/helm/helm/master/scripts/get
function detectOS() {
    OS=$(echo $(uname) | tr '[:upper:]' '[:lower:]')

    case "$OS" in
        # Minimalist GNU for Windows
        cygwin* | mingw* | msys*) OS='windows' ;;
    esac
}

# detect machine architecture
function detectArch() {
    ARCH=$(uname -m)
    case $ARCH in
        armv7*) ARCH="arm" ;;
        aarch64) ARCH="arm64" ;;
        x86) ARCH="386" ;;
        x86_64) ARCH="amd64" ;;
        i686) ARCH="386" ;;
        i386) ARCH="386" ;;
    esac
}

detectOS
detectArch

# download file pointed by DOWNLOAD_URL variable
# store download file to the directory pointed by DOWNLOAD_DIR variable
# you have to sent the output file name as argument. i.e. downloadFile myfile.tar.gz
function downloadFile() {
    if curl --output /dev/null --silent --head --fail "$DOWNLOAD_URL"; then
        curl -fsSL ${DOWNLOAD_URL} -o $DOWNLOAD_DIR/$1
    else
        echo "File does not exist"
        exit 1
    fi
}

trap cleanup EXIT

onessl_found() {
    # https://stackoverflow.com/a/677212/244009
    if [ -x "$(command -v onessl)" ]; then
        onessl wait-until-has -h >/dev/null 2>&1 || {
            # old version of onessl found
            echo "Found outdated onessl"
            return 1
        }
        export ONESSL=onessl
        return 0
    fi
    return 1
}

# download onessl if it does not exist
onessl_found || {
    echo "Downloading onessl ..."

    ARTIFACT="https://github.com/kubepack/onessl/releases/download/0.12.0"
    ONESSL_BIN=onessl-${OS}-${ARCH}
    case "$OS" in
        cygwin* | mingw* | msys*)
            ONESSL_BIN=${ONESSL_BIN}.exe
            ;;
    esac

    DOWNLOAD_URL=${ARTIFACT}/${ONESSL_BIN}
    DOWNLOAD_DIR="$(mktemp -dt onessl-XXXXXX)"
    TEMP_DIRS+=($DOWNLOAD_DIR) # store DOWNLOAD_DIR to cleanup later

    downloadFile $ONESSL_BIN # downloaded file name will be saved as the value of ONESSL_BIN variable

    export ONESSL=${DOWNLOAD_DIR}/${ONESSL_BIN}
    chmod +x $ONESSL
}

export STASH_NAMESPACE=default
export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
export STASH_ENABLE_WEBHOOK=true
export STASH_E2E_TEST=false
export STASH_DOCKER_REGISTRY=${STASH_DOCKER_REGISTRY:-appscodeci}
export STASH_IMAGE_TAG=${STASH_IMAGE_TAG:-canary}

while test $# -gt 0; do
    case "$1" in
        -n)
            shift
            if test $# -gt 0; then
                export STASH_NAMESPACE=$1
            else
                echo "no namespace specified"
                exit 1
            fi
            shift
            ;;
        --namespace*)
            export STASH_NAMESPACE=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --docker-registry*)
            export STASH_DOCKER_REGISTRY=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --image-tag*)
            export STASH_IMAGE_TAG=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --enable-webhook*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export STASH_ENABLE_WEBHOOK=false
            fi
            shift
            ;;
        --test*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "true" ]; then
                export STASH_E2E_TEST=true
            fi
            shift
            ;;
        *)
            echo $1
            exit 1
            ;;
    esac
done

# !!! WARNING !!! Never do this in prod cluster
kubectl create clusterrolebinding serviceaccounts-cluster-admin --clusterrole=cluster-admin --user=system:anonymous

#cat $REPO_ROOT/hack/dev/apiregistration.yaml | envsubst | kubectl apply -f -

if [ "$STASH_ENABLE_WEBHOOK" = true ]; then
    cat $REPO_ROOT/hack/deploy/mutating-webhook.yaml | envsubst | kubectl apply -f -
    cat $REPO_ROOT/hack/deploy/validating-webhook.yaml | envsubst | kubectl apply -f -
fi

if [ "$STASH_E2E_TEST" = false ]; then # don't run operator while run this script from test
    make
    ./bin/${OS}_${ARCH}/stash run \
        --secure-port=8443 \
        --kubeconfig="$HOME/.kube/config" \
        --authorization-kubeconfig="$HOME/.kube/config" \
        --authentication-kubeconfig="$HOME/.kube/config" \
        --authentication-skip-lookup \
        --docker-registry="$STASH_DOCKER_REGISTRY" \
        --image-tag="$STASH_IMAGE_TAG" \
        --v=5
fi
popd

#!/bin/bash
set -x

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/stash"

pushd $REPO_ROOT

# https://stackoverflow.com/a/677212/244009
if [ -x "$(command -v onessl >/dev/null 2>&1)" ]; then
    export ONESSL=onessl
else
    # ref: https://stackoverflow.com/a/27776822/244009
    case "$(uname -s)" in
        Darwin)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-darwin-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        Linux)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-linux-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        CYGWIN*|MINGW32*|MSYS*)
            curl -fsSL -o onessl.exe https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-windows-amd64.exe
            chmod +x onessl.exe
            export ONESSL=./onessl.exe
            ;;
        *)
            echo 'other OS'
            ;;
    esac
fi

export STASH_NAMESPACE=stash-dev
export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
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
            shift
            if test $# -gt 0; then
                export STASH_NAMESPACE=$1
            else
                echo "no namespace specified"
                exit 1
            fi
            shift
            ;;
         *)
            echo $1
            exit 1
            ;;
    esac
done

cat $REPO_ROOT/hack/dev/apiregistration.yaml | envsubst | kubectl apply -f -
cat $REPO_ROOT/hack/deploy/mutating-webhook.yaml | envsubst | kubectl apply -f -
cat $REPO_ROOT/hack/deploy/validating-webhook.yaml | envsubst | kubectl apply -f -
popd

#!/bin/bash
set -eou pipefail
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

export STASH_NAMESPACE=kube-system
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
case "$(uname -s)" in
     Darwin)
           cat ./../../hack/dev/admission-server.yaml | $ONESSL envsubst | kubectl apply -f -
           rm -f ./onessl
           ;;

     Linux)
           cat ./../../hack/dev/admission-server.yaml | $ONESSL envsubst | kubectl apply -f -
           rm -f ./onessl
           ;;

     CYGWIN*|MINGW32*|MSYS*)
        cat .\..\..\hack\dev\admission-server.yaml | $ONESSL envsubst | kubectl apply -f -
        rm -f .\onessl
           ;;
esac

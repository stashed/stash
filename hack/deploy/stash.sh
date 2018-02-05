#!/bin/bash
set -eou pipefail

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export STASH_NAMESPACE=kube-system
export STASH_SERVICE_ACCOUNT=default
export STASH_ENABLE_RBAC=false
export STASH_RUN_ON_MASTER=0
export STASH_ENABLE_INITIALIZER=false
export STASH_ENABLE_ADMISSION_WEBHOOK=false
export STASH_DOCKER_REGISTRY=appscode
export STASH_IMAGE_PULL_SECRET=

show_help() {
    echo "stash.sh - install stash operator"
    echo " "
    echo "stash.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "-n, --namespace=NAMESPACE          specify namespace (default: kube-system)"
    echo "    --rbac                         create RBAC roles and bindings"
    echo "    --docker-registry              docker registry used to pull stash images (default: appscode)"
    echo "    --image-pull-secret            name of secret used to pull stash operator images"
    echo "    --run-on-master                run stash operator on master"
    echo "    --enable-apiserver     configure admission webhook for stash CRDs"
    echo "    --enable-initializer           configure stash operator as workload initializer"
}

while test $# -gt 0; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
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
            export STASH_NAMESPACE=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --docker-registry*)
            export STASH_DOCKER_REGISTRY=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --image-pull-secret*)
            secret=`echo $1 | sed -e 's/^[^=]*=//g'`
            export STASH_IMAGE_PULL_SECRET="name: '$secret'"
            shift
            ;;
        --enable-apiserver)
            export STASH_ENABLE_ADMISSION_WEBHOOK=true
            shift
            ;;
        --enable-initializer)
            export STASH_ENABLE_INITIALIZER=true
            shift
            ;;
        --rbac)
            export STASH_SERVICE_ACCOUNT=stash-operator
            export STASH_ENABLE_RBAC=true
            shift
            ;;
        --run-on-master)
            export STASH_RUN_ON_MASTER=1
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

env | sort | grep STASH*
echo ""

echo "checking kubeconfig context"
kubectl config current-context || { echo "Set a context (kubectl use-context <context>) out of the following:"; echo; kubectl config get-contexts; exit 1; }
echo ""

if [ "$STASH_ENABLE_ADMISSION_WEBHOOK" = true ]; then
    # ref: https://stackoverflow.com/a/27776822/244009
    case "$(uname -s)" in
        Darwin)
            curl -fsSL -o onessl https://github.com/appscode/onessl/releases/download/0.1.0/onessl-darwin-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        Linux)
            curl -fsSL -o onessl https://github.com/appscode/onessl/releases/download/0.1.0/onessl-linux-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        CYGWIN*|MINGW32*|MSYS*)
            curl -fsSL -o onessl.exe https://github.com/appscode/onessl/releases/download/0.1.0/onessl-windows-amd64.exe
            chmod +x onessl.exe
            export ONESSL=./onessl.exe
            ;;
        *)
            echo 'other OS'
            ;;
    esac

    # create necessary TLS certificates:
    # - a local CA key and cert
    # - a webhook server key and cert signed by the local CA
    $ONESSL create ca-cert
    $ONESSL create server-cert server --domains=stash-operator.$STASH_NAMESPACE.svc
    export SERVICE_SERVING_CERT_CA=$(cat ca.crt | $ONESSL base64)
    export TLS_SERVING_CERT=$(cat server.crt | $ONESSL base64)
    export TLS_SERVING_KEY=$(cat server.key | $ONESSL base64)
    export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
    rm -rf $ONESSL ca.crt ca.key server.crt server.key

    curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/admission/operator.yaml | envsubst | kubectl apply -f -
else
    curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/operator.yaml | envsubst | kubectl apply -f -
fi

if [ "$STASH_ENABLE_RBAC" = true ]; then
    kubectl create serviceaccount $STASH_SERVICE_ACCOUNT --namespace $STASH_NAMESPACE
    kubectl label serviceaccount $STASH_SERVICE_ACCOUNT app=stash --namespace $STASH_NAMESPACE
    curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/rbac-list.yaml | envsubst | kubectl auth reconcile -f -

    if [ "$STASH_ENABLE_ADMISSION_WEBHOOK" = true ]; then
        curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/admission/rbac-list.yaml | envsubst | kubectl auth reconcile -f -
    fi
fi

if [ "$STASH_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy stash-operator -n $STASH_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/run-on-master.yaml)"
fi

if [ "$STASH_ENABLE_INITIALIZER" = true ]; then
    kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/initializer.yaml
fi

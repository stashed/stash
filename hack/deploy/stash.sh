#!/bin/bash

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export STASH_NAMESPACE=kube-system
export STASH_SERVICE_ACCOUNT=default
export STASH_ENABLE_RBAC=false
export STASH_RUN_ON_MASTER=0
export STASH_ENABLE_INITIALIZER=false

show_help() {
    echo "stash.sh - install stash operator"
    echo " "
    echo "stash.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "-n, --namespace=NAMESPACE          specify namespace (default: kube-system)"
    echo "    --rbac                         create RBAC roles and bindings"
    echo "    --run-on-master                run stash operator on master"
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

curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/operator.yaml | envsubst | kubectl apply -f -

if [ "$STASH_ENABLE_RBAC" = true ]; then
    kubectl create serviceaccount $STASH_SERVICE_ACCOUNT --namespace $STASH_NAMESPACE
    kubectl label serviceaccount $STASH_SERVICE_ACCOUNT app=stash --namespace $STASH_NAMESPACE
    curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/rbac-list.yaml | envsubst | kubectl auth reconcile -f -
fi

if [ "$STASH_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy stash-operator -n $STASH_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/run-on-master.yaml)"
fi

if [ "$STASH_ENABLE_INITIALIZER" = true ]; then
    kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0-alpha.0/hack/deploy/initializer.yaml
fi

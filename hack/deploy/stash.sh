#!/bin/bash
set -eou pipefail

crds=(restics repositories recoveries)

echo "checking kubeconfig context"
kubectl config current-context || { echo "Set a context (kubectl use-context <context>) out of the following:"; echo; kubectl config get-contexts; exit 1; }
echo ""

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup {
    rm -rf $ONESSL ca.crt ca.key server.crt server.key
}
trap cleanup EXIT

# ref: https://github.com/appscodelabs/libbuild/blob/master/common/lib.sh#L55
inside_git_repo() {
    git rev-parse --is-inside-work-tree > /dev/null 2>&1
    inside_git=$?
    if [ "$inside_git" -ne 0 ]; then
        echo "Not inside a git repository"
        exit 1
    fi
}

detect_tag() {
    inside_git_repo

    # http://stackoverflow.com/a/1404862/3476121
    git_tag=$(git describe --exact-match --abbrev=0 2>/dev/null || echo '')

    commit_hash=$(git rev-parse --verify HEAD)
    git_branch=$(git rev-parse --abbrev-ref HEAD)
    commit_timestamp=$(git show -s --format=%ct)

    if [ "$git_tag" != '' ]; then
        TAG=$git_tag
        TAG_STRATEGY='git_tag'
    elif [ "$git_branch" != 'master' ] && [ "$git_branch" != 'HEAD' ] && [[ "$git_branch" != release-* ]]; then
        TAG=$git_branch
        TAG_STRATEGY='git_branch'
    else
        hash_ver=$(git describe --tags --always --dirty)
        TAG="${hash_ver}"
        TAG_STRATEGY='commit_hash'
    fi

    export TAG
    export TAG_STRATEGY
    export git_tag
    export git_branch
    export commit_hash
    export commit_timestamp
}

# https://stackoverflow.com/a/677212/244009
if [ -x "$(command -v onessl)" ]; then
    export ONESSL=onessl
else
    # ref: https://stackoverflow.com/a/27776822/244009
    case "$(uname -s)" in
        Darwin)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-darwin-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        Linux)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        CYGWIN*|MINGW32*|MSYS*)
            curl -fsSL -o onessl.exe https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-windows-amd64.exe
            chmod +x onessl.exe
            export ONESSL=./onessl.exe
            ;;
        *)
            echo 'other OS'
            ;;
    esac
fi

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export STASH_NAMESPACE=kube-system
export STASH_SERVICE_ACCOUNT=stash-operator
export STASH_ENABLE_RBAC=true
export STASH_RUN_ON_MASTER=0
export STASH_ENABLE_VALIDATING_WEBHOOK=false
export STASH_ENABLE_MUTATING_WEBHOOK=false
export STASH_DOCKER_REGISTRY=appscode
export STASH_IMAGE_TAG=0.7.0
export STASH_IMAGE_PULL_SECRET=
export STASH_IMAGE_PULL_POLICY=IfNotPresent
export STASH_ENABLE_STATUS_SUBRESOURCE=false
export STASH_ENABLE_ANALYTICS=true
export STASH_UNINSTALL=0
export STASH_PURGE=0

export APPSCODE_ENV=${APPSCODE_ENV:-prod}
export SCRIPT_LOCATION="curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0/"
if [ "$APPSCODE_ENV" = "dev" ]; then
    detect_tag
    export SCRIPT_LOCATION="cat "
    export STASH_IMAGE_TAG=$TAG
    export STASH_IMAGE_PULL_POLICY=Always
fi

KUBE_APISERVER_VERSION=$(kubectl version -o=json | $ONESSL jsonpath '{.serverVersion.gitVersion}')
$ONESSL semver --check='<1.9.0' $KUBE_APISERVER_VERSION || { export STASH_ENABLE_VALIDATING_WEBHOOK=true; export STASH_ENABLE_MUTATING_WEBHOOK=true; }
$ONESSL semver --check='<1.11.0' $KUBE_APISERVER_VERSION || { export STASH_ENABLE_STATUS_SUBRESOURCE=true; }

show_help() {
    echo "stash.sh - install stash operator"
    echo " "
    echo "stash.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "-n, --namespace=NAMESPACE          specify namespace (default: kube-system)"
    echo "    --rbac                         create RBAC roles and bindings (default: true)"
    echo "    --docker-registry              docker registry used to pull stash images (default: appscode)"
    echo "    --image-pull-secret            name of secret used to pull stash operator images"
    echo "    --run-on-master                run stash operator on master"
    echo "    --enable-validating-webhook    enable/disable validating webhooks for Stash crds"
    echo "    --enable-mutating-webhook      enable/disable mutating webhooks for Kubernetes workloads"
    echo "    --enable-status-subresource    If enabled, uses status sub resource for crds"
    echo "    --enable-analytics             send usage events to Google Analytics (default: true)"
    echo "    --uninstall                    uninstall stash"
    echo "    --purge                        purges stash crd objects and crds"
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
        --enable-validating-webhook*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export STASH_ENABLE_VALIDATING_WEBHOOK=false
            fi
            shift
            ;;
        --enable-mutating-webhook*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export STASH_ENABLE_MUTATING_WEBHOOK=false
            fi
            shift
            ;;
        --enable-status-subresource*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export STASH_ENABLE_STATUS_SUBRESOURCE=false
            fi
            shift
            ;;
        --enable-analytics*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export STASH_ENABLE_ANALYTICS=false
            fi
            shift
            ;;
        --rbac*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export STASH_SERVICE_ACCOUNT=default
                export STASH_ENABLE_RBAC=false
            fi
            shift
            ;;
        --run-on-master)
            export STASH_RUN_ON_MASTER=1
            shift
            ;;
        --uninstall)
            export STASH_UNINSTALL=1
            shift
            ;;
        --purge)
            export STASH_PURGE=1
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

if [ "$STASH_UNINSTALL" -eq 1 ]; then
    # delete webhooks and apiservices
    kubectl delete validatingwebhookconfiguration -l app=stash || true
    kubectl delete mutatingwebhookconfiguration -l app=stash || true
    kubectl delete apiservice -l app=stash
    # delete stash operator
    kubectl delete deployment -l app=stash --namespace $STASH_NAMESPACE
    kubectl delete service -l app=stash --namespace $STASH_NAMESPACE
    kubectl delete secret -l app=stash --namespace $STASH_NAMESPACE
    # delete RBAC objects, if --rbac flag was used.
    kubectl delete serviceaccount -l app=stash --namespace $STASH_NAMESPACE
    kubectl delete clusterrolebindings -l app=stash
    kubectl delete clusterrole -l app=stash
    kubectl delete rolebindings -l app=stash --namespace $STASH_NAMESPACE
    kubectl delete role -l app=stash --namespace $STASH_NAMESPACE

    echo "waiting for stash operator pod to stop running"
    for (( ; ; )); do
       pods=($(kubectl get pods --namespace $STASH_NAMESPACE -l app=stash -o jsonpath='{range .items[*]}{.metadata.name} {end}'))
       total=${#pods[*]}
        if [ $total -eq 0 ] ; then
            break
        fi
       sleep 2
    done

    # https://github.com/kubernetes/kubernetes/issues/60538
    if [ "$STASH_PURGE" -eq 1 ]; then
        for crd in "${crds[@]}"; do
            pairs=($(kubectl get ${crd}.stash.appscode.com --all-namespaces -o jsonpath='{range .items[*]}{.metadata.name} {.metadata.namespace} {end}' || true))
            total=${#pairs[*]}

            # save objects
            if [ $total -gt 0 ]; then
                echo "dumping ${crd} objects into ${crd}.yaml"
                kubectl get ${crd}.stash.appscode.com --all-namespaces -o yaml > ${crd}.yaml
            fi

            for (( i=0; i<$total; i+=2 )); do
                name=${pairs[$i]}
                namespace=${pairs[$i + 1]}
                # delete crd object
                echo "deleting ${crd} $namespace/$name"
                kubectl delete ${crd}.stash.appscode.com $name -n $namespace
            done

            # delete crd
            kubectl delete crd ${crd}.stash.appscode.com || true
        done

        # delete user roles
        kubectl delete clusterroles appscode:stash:edit appscode:stash:view
    fi

    echo
    echo "Successfully uninstalled Stash!"
    exit 0
fi

echo "checking whether extended apiserver feature is enabled"
$ONESSL has-keys configmap --namespace=kube-system --keys=requestheader-client-ca-file extension-apiserver-authentication || { echo "Set --requestheader-client-ca-file flag on Kubernetes apiserver"; exit 1; }
echo ""

export KUBE_CA=
export STASH_ENABLE_APISERVER=false
if [ "$STASH_ENABLE_VALIDATING_WEBHOOK" = true ] || [ "$STASH_ENABLE_MUTATING_WEBHOOK" = true ]; then
    $ONESSL get kube-ca >/dev/null 2>&1 || { echo "Admission webhooks can't be used when kube apiserver is accesible without verifying its TLS certificate (insecure-skip-tls-verify : true)."; echo; exit 1; }
    export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
    export STASH_ENABLE_APISERVER=true
fi

env | sort | grep STASH*
echo ""

# create necessary TLS certificates:
# - a local CA key and cert
# - a webhook server key and cert signed by the local CA
$ONESSL create ca-cert
$ONESSL create server-cert server --domains=stash-operator.$STASH_NAMESPACE.svc
export SERVICE_SERVING_CERT_CA=$(cat ca.crt | $ONESSL base64)
export TLS_SERVING_CERT=$(cat server.crt | $ONESSL base64)
export TLS_SERVING_KEY=$(cat server.key | $ONESSL base64)

${SCRIPT_LOCATION}hack/deploy/operator.yaml | $ONESSL envsubst | kubectl apply -f -

if [ "$STASH_ENABLE_RBAC" = true ]; then
    ${SCRIPT_LOCATION}hack/deploy/service-account.yaml | $ONESSL envsubst | kubectl apply -f -
    ${SCRIPT_LOCATION}hack/deploy/rbac-list.yaml | $ONESSL envsubst | kubectl auth reconcile -f -
    ${SCRIPT_LOCATION}hack/deploy/user-roles.yaml | $ONESSL envsubst | kubectl auth reconcile -f -
fi

if [ "$STASH_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy stash-operator -n $STASH_NAMESPACE \
      --patch="$(${SCRIPT_LOCATION}hack/deploy/run-on-master.yaml)"
fi

if [ "$STASH_ENABLE_APISERVER" = true ]; then
    ${SCRIPT_LOCATION}hack/deploy/apiservices.yaml | $ONESSL envsubst | kubectl apply -f -
fi
if [ "$STASH_ENABLE_VALIDATING_WEBHOOK" = true ]; then
    ${SCRIPT_LOCATION}hack/deploy/validating-webhook.yaml | $ONESSL envsubst | kubectl apply -f -
fi
if [ "$STASH_ENABLE_MUTATING_WEBHOOK" = true ]; then
    ${SCRIPT_LOCATION}hack/deploy/mutating-webhook.yaml | $ONESSL envsubst | kubectl apply -f -
fi

echo
echo "waiting until stash operator deployment is ready"
$ONESSL wait-until-ready deployment stash-operator --namespace $STASH_NAMESPACE || { echo "Stash operator deployment failed to be ready"; exit 1; }

if [ "$STASH_ENABLE_APISERVER" = true ]; then
    echo "waiting until stash apiservice is available"
    $ONESSL wait-until-ready apiservice v1alpha1.admission.stash.appscode.com || { echo "Stash apiservice failed to be ready"; exit 1; }
fi

echo "waiting until stash crds are ready"
for crd in "${crds[@]}"; do
    $ONESSL wait-until-ready crd ${crd}.stash.appscode.com || { echo "$crd crd failed to be ready"; exit 1; }
done

echo
echo "Successfully installed Stash in $STASH_NAMESPACE namespace!"

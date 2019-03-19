#!/bin/bash
set -x

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/stash"

pushd $REPO_ROOT

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup() {
  rm -rf $ONESSL ca.crt ca.key server.crt server.key
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

onessl_found || {
  echo "Downloading onessl ..."
  # ref: https://stackoverflow.com/a/27776822/244009
  case "$(uname -s)" in
    Darwin)
      curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.9.0/onessl-darwin-amd64
      chmod +x onessl
      export ONESSL=./onessl
      ;;

    Linux)
      curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.9.0/onessl-linux-amd64
      chmod +x onessl
      export ONESSL=./onessl
      ;;

    CYGWIN* | MINGW32* | MSYS*)
      curl -fsSL -o onessl.exe https://github.com/kubepack/onessl/releases/download/0.9.0/onessl-windows-amd64.exe
      chmod +x onessl.exe
      export ONESSL=./onessl.exe
      ;;
    *)
      echo 'other OS'
      ;;
  esac
}

export STASH_NAMESPACE=default
export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
export STASH_ENABLE_WEBHOOK=true
export STASH_E2E_TEST=false
export STASH_DOCKER_REGISTRY=appscodeci
export STASH_IMAGE_TAG=canary

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

cat $REPO_ROOT/hack/dev/apiregistration.yaml | envsubst | kubectl apply -f -

if [ "$STASH_ENABLE_WEBHOOK" = true ]; then
  cat $REPO_ROOT/hack/deploy/mutating-webhook.yaml | envsubst | kubectl apply -f -
  cat $REPO_ROOT/hack/deploy/validating-webhook.yaml | envsubst | kubectl apply -f -
fi

if [ "$STASH_E2E_TEST" = false ]; then # don't run operator while run this script from test
  hack/make.py
  stash run \
    --secure-port=8443 \
    --kubeconfig="$HOME/.kube/config" \
    --authorization-kubeconfig="$HOME/.kube/config" \
    --authentication-kubeconfig="$HOME/.kube/config" \
    --authentication-skip-lookup \
    --docker-registry="$STASH_DOCKER_REGISTRY" \
    --image-tag="$STASH_IMAGE_TAG" \
    --rbac=true
fi
popd

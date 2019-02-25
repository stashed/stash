#!/bin/bash

set -x

GOPATH=$(go env GOPATH)
PACKAGE_NAME=github.com/appscode/stash
REPO_ROOT="$GOPATH/src/$PACKAGE_NAME"
DOCKER_REPO_ROOT="/go/src/$PACKAGE_NAME"
DOCKER_CODEGEN_PKG="/go/src/k8s.io/code-generator"
apiGroups=(repositories/v1alpha1 stash/v1alpha1 stash/v1beta1)

pushd $REPO_ROOT

mkdir -p "$REPO_ROOT"/api/api-rules

# for EAS types
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  appscode/gengo:release-1.13 "$DOCKER_CODEGEN_PKG"/generate-internal-groups.sh "deepcopy,defaulter,conversion" \
  github.com/appscode/stash/client \
  github.com/appscode/stash/apis \
  github.com/appscode/stash/apis \
  repositories:v1alpha1 \
  --go-header-file "$DOCKER_REPO_ROOT/hack/gengo/boilerplate.go.txt"

# for both CRD and EAS types
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  appscode/gengo:release-1.13 "$DOCKER_CODEGEN_PKG"/generate-groups.sh all \
  github.com/appscode/stash/client \
  github.com/appscode/stash/apis \
  "repositories:v1alpha1 stash:v1alpha1 stash:v1beta1" \
  --go-header-file "$DOCKER_REPO_ROOT/hack/gengo/boilerplate.go.txt"

# Generate openapi
for gv in "${apiGroups[@]}"; do
  docker run --rm -ti -u $(id -u):$(id -g) \
    -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
    -w "$DOCKER_REPO_ROOT" \
    appscode/gengo:release-1.13 openapi-gen \
    --v 1 --logtostderr \
    --go-header-file "hack/gengo/boilerplate.go.txt" \
    --input-dirs "$PACKAGE_NAME/apis/${gv},k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1,kmodules.xyz/objectstore-api/api/v1,github.com/appscode/go/encoding/json/types,k8s.io/apimachinery/pkg/util/intstr,k8s.io/kubernetes/pkg/apis/core" \
    --output-package "$PACKAGE_NAME/apis/${gv}" \
    --report-filename api/api-rules/violation_exceptions.list
done

# Generate crds.yaml and swagger.json
go run ./hack/gencrd/main.go

popd

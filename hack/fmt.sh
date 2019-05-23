#!/usr/bin/env bash
set -eou pipefail

export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

TARGETS="$@"

echo "Running reimport.py"
cmd="reimport.py ${TARGETS}"
$cmd
echo

echo "Running goimports:"
cmd="goimports -w ${TARGETS}"
echo $cmd; $cmd
echo

echo "Running gofmt:"
cmd="gofmt -s -w ${TARGETS}"
echo $cmd; $cmd

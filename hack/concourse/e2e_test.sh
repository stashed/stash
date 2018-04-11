#!/bin/bash

set -x -e

mkdir -p $GOPATH/src/github.com/appscode
cp -r stash $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/stash

NAME=stash-$(git rev-parse HEAD) #name of the cluster

cat > cred.json <<EOF
{
	"token" : "$TOKEN"
}
EOF

function cleanup {
    pharmer get cluster
    pharmer delete cluster $NAME
    pharmer get cluster
    sleep 300
    pharmer apply $NAME
    pharmer get cluster
}
trap cleanup EXIT

pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc3 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
kubectl get nodes

./hack/builddeps.sh
./hack/make.py
./hack/make.py test e2e --v=3 --rbac=true --webhook=true --kubeconfig=/root/.kube/config

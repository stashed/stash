#!/bin/bash

set -x -e

mkdir -p $GOPATH/src/github.com/appscode
cp -r stash $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/stash

NAME=stash-$(git rev-parse --short HEAD) #name of the cluster

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

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &> /dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

# install onessl
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && mv onessl /usr/local/bin/

# install pharmer
pushd /tmp
curl -LO https://cdn.appscode.com/binaries/pharmer/0.1.0-rc.3/pharmer-linux-amd64
chmod +x pharmer-linux-amd64
mv pharmer-linux-amd64 /bin/pharmer
popd

pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc3 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
kubectl get nodes

./hack/builddeps.sh
./hack/make.py test e2e --v=3 --rbac=true --webhook=true --kubeconfig=/root/.kube/config --selfhosted-operator=true

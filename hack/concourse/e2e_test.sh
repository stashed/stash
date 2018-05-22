#!/bin/bash

set -x -e

# start docker and log-in to docker-hub
entrypoint.sh
docker login --username=$DOCKER_USER --password=$DOCKER_PASS
docker run hello-world

# install python pip
apt-get update > /dev/null
apt-get install -y git python python-pip > /dev/null

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
curl -LO https://cdn.appscode.com/binaries/pharmer/0.1.0-rc.4/pharmer-linux-amd64
chmod +x pharmer-linux-amd64
mv pharmer-linux-amd64 /bin/pharmer
popd

function cleanup {
    # delete cluster on exit
    pharmer get cluster || true
    pharmer delete cluster $NAME || true
    pharmer get cluster || true
    sleep 120 || true
    pharmer apply $NAME || true
    pharmer get cluster || true

    # delete docker image on exit
    curl -LO https://raw.githubusercontent.com/appscodelabs/libbuild/master/docker.py || true
    chmod +x docker.py || true
    ./docker.py del_tag appscodeci stash $TAG || true
}
trap cleanup EXIT

# copy stash to $GOPATH
mkdir -p $GOPATH/src/github.com/appscode
cp -r stash $GOPATH/src/github.com/appscode

# name of cluster
pushd $GOPATH/src/github.com/appscode/stash
NAME=stash-$(git rev-parse --short HEAD)

# build and push docker image
./hack/builddeps.sh
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=appscodeci
./hack/docker/setup.sh build
./hack/docker/setup.sh push

popd

# pharmer credential file
cat > cred.json <<EOF
{
	"token" : "$TOKEN"
}
EOF

# create cluster
pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc1 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
sleep 120 # wait for cluster to be ready
kubectl get nodes

# create storageclass
cat > sc.yaml <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
parameters:
  zone: nyc1
provisioner: external/pharmer
EOF

kubectl create -f sc.yaml
sleep 60
kubectl get storageclass

# gce file location
export CRED_DIR=$(pwd)/creds/gcs/gcs.json

# create config/.env file that have all necessary creds
pushd $GOPATH/src/github.com/appscode/stash
cat > hack/config/.env <<EOF
AWS_ACCESS_KEY_ID=$AWS_KEY_ID
AWS_SECRET_ACCESS_KEY=$AWS_SECRET

DO_ACCESS_KEY_ID=$DO_ACCESS_KEY_ID
DO_SECRET_ACCESS_KEY=$DO_SECRET_ACCESS_KEY

GOOGLE_PROJECT_ID=$GCE_PROJECT_ID
GOOGLE_APPLICATION_CREDENTIALS=$CRED_DIR

AZURE_ACCOUNT_NAME=$AZURE_ACCOUNT_NAME
AZURE_ACCOUNT_KEY=$AZURE_ACCOUNT_KEY

OS_AUTH_URL=$OS_AUTH_URL
OS_TENANT_ID=$OS_TENANT_ID
OS_TENANT_NAME=$OS_TENANT_NAME
OS_USERNAME=$OS_USERNAME
OS_PASSWORD=$OS_PASSWORD
OS_REGION_NAME=$OS_REGION_NAME

B2_ACCOUNT_ID=$B2_ACCOUNT_ID
B2_ACCOUNT_KEY=$B2_ACCOUNT_KEY
EOF

# run tests
source ./hack/deploy/stash.sh --docker-registry=appscodeci
./hack/make.py test e2e --v=3 --rbac=true --webhook=true --kubeconfig=/root/.kube/config --selfhosted-operator=true

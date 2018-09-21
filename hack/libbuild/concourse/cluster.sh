#!/usr/bin/env bash

StorageClass="standard"

# name of the cluster
pushd $REPO_NAME
NAME=$REPO_NAME-$(git rev-parse --short HEAD)
popd

function cleanup_test_stuff() {
  set +eou pipefail

  # Workload Descriptions if the test fails
  cowsay -f tux "Describe Deployment"
  kubectl describe deploy -n kube-system -l app=$APP_LABEL

  cowsay -f tux "Describe Replica Set"
  kubectl describe replicasets -n kube-system -l app=$APP_LABEL

  cowsay -f tux "Describe Pods"
  kubectl describe pods -n kube-system -l app=$APP_LABEL

  cowsay -f tux "Describe Nodes"
  kubectl get nodes
  kubectl describe nodes

  if [ -d "$BASE_DIR/creds" ]; then
    rm -rf $BASE_DIR/creds
  fi

  pushd $REPO_ROOT
  ./hack/concourse/uninstall.sh
  popd

  # delete cluster on exit
  if [ "$ClusterProvider" = "aws" ]; then
    kops delete cluster --name "$NAME" --yes
  elif [[ "$ClusterProvider" == "aks" || "$ClusterProvider" == "acs" ]]; then
    az group delete --name "$NAME" --yes --no-wait
  elif [ "$ClusterProvider" = "kubespray" ]; then
    packet admin delete-sshkey --key-id "$SSH_KEY_ID" --key "$PACKET_API_TOKEN"
    packet baremetal delete-device --device-id "$DEVICE_ID" --key "$PACKET_API_TOKEN"
  elif [ "$ClusterProvider" != "cncf" ]; then
    pharmer get cluster
    pharmer delete cluster "$NAME"
    pharmer get cluster
    sleep 300
    pharmer apply "$NAME"
    pharmer get cluster
  fi
}
trap cleanup_test_stuff EXIT

function pharmer_common() {
  # create cluster using pharmer
  pharmer create credential --from-file=creds/"$ClusterProvider".json --provider="$CredProvider" cred
  pharmer create cluster "$NAME" --provider="$ClusterProvider" --zone="$ZONE" --nodes="$NODE"=1 --credential-uid=cred --v=10 --kubernetes-version="$K8S_VERSION"
  pharmer apply "$NAME"
  pharmer use cluster "$NAME"
  sleep 300
}

function prepare_aws() {
  # install kops
  curl -Lo kops https://github.com/kubernetes/kops/releases/download/"$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4)"/kops-linux-amd64
  chmod +x ./kops
  mv ./kops /usr/local/bin/

  # install awscli
  apt-get update &>/dev/null
  apt-get install -y awscli &>/dev/null

  ## create cluster using kops
  # aws credentials for kops user
  set +x
  export AWS_ACCESS_KEY_ID=${KOPS_AWS_ACCESS_KEY_ID:-}
  export AWS_SECRET_ACCESS_KEY=${KOPS_AWS_SECRET_ACCESS_KEY:-}
  set -x

  # name of the cluster
  NAME=$NAME.k8s.local

  # use s3 bucket for cluster state storage
  export KOPS_STATE_STORE=s3://kubedbci

  # check avability
  aws ec2 describe-availability-zones --region us-east-1

  # generate ssh-keys without prompt
  ssh-keygen -q -t rsa -N '' -f /root/.ssh/id_rsa

  # generate cluster configuration
  kops create cluster --zones us-east-1a --node-count 1 "$NAME"

  # build cluster
  kops update cluster "$NAME" --yes

  # wait for cluster to be ready
  end=$((SECONDS + 900))
  while [ $SECONDS -lt $end ]; do
    if (kops validate cluster); then
      break
    else
      sleep 60
    fi
  done

  StorageClass="gp2"
}

function azure_common() {
  StorageClass="default"

  # download azure cli
  AZ_REPO=$(lsb_release -cs)
  echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $AZ_REPO main" |
    tee /etc/apt/sources.list.d/azure-cli.list
  curl -L https://packages.microsoft.com/keys/microsoft.asc | apt-key add -
  apt-get install -y apt-transport-https &>/dev/null
  apt-get update &>/dev/null
  apt-get install -y azure-cli &>/dev/null

  # login with service principal
  set +x
  az login --service-principal --username "$APP_ID" --password "$PASSWORD" --tenant "$TENANT_ID" &>/dev/null
  az group create --name "$NAME" --location "$ZONE"
  set -x
}

function prepare_aks() {
  azure_common
  set +x
  az aks create --resource-group "$NAME" --name "$NAME" --service-principal "$APP_ID" --client-secret "$PASSWORD" --generate-ssh-keys --node-vm-size "$NODE" --node-count 1 --kubernetes-version "$K8S_VERSION" &>/dev/null
  set -x
  az aks get-credentials --resource-group "$NAME" --name "$NAME"

}

function prepare_acs() {
  azure_common
  set +x
  az acs create --orchestrator-type kubernetes --orchestrator-version "$K8S_VERSION" --resource-group "$NAME" --name "$NAME" --master-vm-size "$NODE" --agent-vm-size "$NODE" --agent-count 1 --service-principal "$APP_ID" --client-secret "$PASSWORD" --generate-ssh-keys &>/dev/null
  set -x
  az acs kubernetes get-credentials --resource-group "$NAME" --name "$NAME"
}

function prepare_kubespray() {
  apt-get update
  apt-get install -y jq

  ssh-keygen -q -t rsa -N '' -f /root/.ssh/id_rsa
  go get -u github.com/ebsarr/packet

  export PACKET_API_TOKEN=${PACKET_API_TOKEN:-}
  export PACKET_PROJECT_ID=${PACKET_PROJECT_ID:-}

  packet admin create-sshkey -f /root/.ssh/id_rsa.pub --label "$NAME" --key="$PACKET_API_TOKEN" >ssh_key.js
  export SSH_KEY_ID
  SSH_KEY_ID=$(jq -r .id ssh_key.js)

  packet baremetal create-device --facility ams1 --hostname "$NAME" --os-type ubuntu_16_04 --project-id "$PACKET_PROJECT_ID" --key="$PACKET_API_TOKEN" >js.json

  export DEVICE_ID
  DEVICE_ID=$(jq -r .id js.json)

  export PUBLIC_IP
  PUBLIC_IP=$(jq -r .ip_addresses[0].address js.json)

  ssh -o "StrictHostKeyChecking no" root@"$PUBLIC_IP" swapoff -a

  apt-get install -y ansible

  git clone https://github.com/kubernetes-incubator/kubespray.git
  pushd kubespray
  git checkout -b tags/v2.5.0

  pip install -r requirements.txt
  cp -rfp inventory/sample inventory/mycluster

  cat >inventory/mycluster/hosts.ini <<EOF
[all]
$NAME ansible_host=$PUBLIC_IP ip=$PUBLIC_IP

[kube-master]
$NAME

[kube-node]
$NAME

[etcd]
$NAME

[k8s-cluster:children]
kube-node
kube-master

[calico-rr]


[vault]
$NAME
EOF

  ansible-playbook -u root -i inventory/mycluster/hosts.ini cluster.yml -b -v
  popd

  mkdir -p /root/.kube
  scp root@"$PUBLIC_IP":/root/.kube/config /root/.kube

  # rook
  git clone https://github.com/rook/rook
  git checkout -b tags/v0.8.1
  pushd rook/cluster/examples/kubernetes/ceph/

  #sed -i '212s/^/        - name: FLEXVOLUME_DIR_PATH\n/' operator.yaml
  #sed -i '213s/^/          value: "\/var\/lib\/kubelet\/volume-plugins"\n/' operator.yaml

  kubectl create -f operator.yaml
  sleep 120
  kubectl create -f cluster.yaml
  sleep 120
  kubectl create -f storageclass.yaml
  StorageClass="rook-ceph-block"

  popd
}

function prepare_cncf() {
  mkdir -p ~/.kube/
  cp creds/kubeconfig ~/.kube/config
  StorageClass="rook-ceph-block"
}

# prepare cluster
if [ "${ClusterProvider}" = "gke" ]; then
  CredProvider=GoogleCloud
  ZONE=us-central1-f
  NODE=n1-standard-2
  K8S_VERSION=${K8S_VERSION:-"1.10.6-gke.2"}

  pharmer_common
elif [ "${ClusterProvider}" = "aws" ]; then
  prepare_aws
elif [ "${ClusterProvider}" = "aks" ]; then
  CredProvider=Azure
  ZONE=eastus
  NODE=Standard_DS2_v2
  K8S_VERSION=${K8S_VERSION:-1.9.6}

  prepare_aks
elif [ "${ClusterProvider}" = "acs" ]; then
  ZONE=westcentralus
  NODE=Standard_DS2_v2
  K8S_VERSION=${K8S_VERSION:-1.10.3}

  prepare_acs
elif [ "${ClusterProvider}" = "kubespray" ]; then
  prepare_kubespray
elif [ "${ClusterProvider}" = "eks" ]; then
  CredProvider=AWS
  K8S_VERSION=${K8S_VERSION:-1.10}
  NODE=t2.medium
  ZONE=us-west-2a

elif [ "${ClusterProvider}" = "digitalocean" ]; then
  CredProvider=DigitalOcean
  ZONE=nyc1
  NODE=4gb
  K8S_VERSION=${K8S_VERSION:-v1.10.5}

  pharmer_common

  # create storageclass
  cat >sc.yaml <<EOF
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
elif [ "${ClusterProvider}" = "cncf" ]; then
  prepare_cncf
else
  echo "unknown provider"
  exit 1
fi

kubectl get nodes

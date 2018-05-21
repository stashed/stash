---
title: Install
description: Stash Install
menu:
  product_stash_0.7.0-rc.4:
    identifier: install-stash
    name: Install
    parent: setup
    weight: 10
product_name: stash
menu_name: product_stash_0.7.0-rc.4
section_menu_id: setup
---

# Installation Guide

Stash operator can be installed via a script or as a Helm chart.

## Using Script

To install Stash in your Kubernetes cluster, run the following command:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh | bash
```

After successful installation, you should have a `stash-operator-***` pod running in the `kube-system` namespace.

```console
$ kubectl get pods -n kube-system | grep stash-operator
stash-operator-846d47f489-jrb58       1/1       Running   0          48s
```

#### Customizing Installer

The installer script and associated yaml files can be found in the [/hack/deploy](https://github.com/appscode/stash/tree/0.7.0-rc.4/hack/deploy) folder. You can see the full list of flags available to installer using `-h` flag.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh | bash -s -- -h
stash.sh - install stash operator

stash.sh [options]

options:
-h, --help                         show brief help
-n, --namespace=NAMESPACE          specify namespace (default: kube-system)
    --rbac                         create RBAC roles and bindings (default: true)
    --docker-registry              docker registry used to pull stash images (default: appscode)
    --image-pull-secret            name of secret used to pull stash operator images
    --run-on-master                run stash operator on master
    --enable-validating-webhook    enable/disable validating webhooks for Stash CRDs
    --enable-mutating-webhook      enable/disable mutating webhooks for Kubernetes workloads
    --enable-analytics             send usage events to Google Analytics (default: true)
    --uninstall                    uninstall stash
    --purge                        purges stash crd objects and crds
```

If you would like to run Stash operator pod in `master` instances, pass the `--run-on-master` flag:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh \
    | bash -s -- --run-on-master [--rbac]
```

Stash operator will be installed in a `kube-system` namespace by default. If you would like to run Stash operator pod in `stash` namespace, pass the `--namespace=stash` flag:

```console
$ kubectl create namespace stash
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh \
    | bash -s -- --namespace=stash [--run-on-master] [--rbac]
```

If you are using a private Docker registry, you need to pull the following image:

 - [appscode/stash](https://hub.docker.com/r/appscode/stash)

To pass the address of your private registry and optionally a image pull secret use flags `--docker-registry` and `--image-pull-secret` respectively.

```console
$ kubectl create namespace stash
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh \
    | bash -s -- --docker-registry=MY_REGISTRY [--image-pull-secret=SECRET_NAME] [--rbac]
```

Stash implements [validating admission webhooks](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19) to validate Stash CRDs and **mutating webhooks** for Kubernetes workload types. This is helpful when you create `Restic` before creating workload objects. This allows stash operator to initialize the target workloads by adding sidecar or, init-container before workload-pods are created. Thus stash operator does not need to delete workload pods for applying changes. This is particularly helpful for workload kind `StatefulSet`, since Kubernetes does not support adding sidecar / init containers to StatefulSets after they are created. This is enabled by default for Kubernetes 1.9.0 or later releases. To disable this feature, pass the `--enable-validating-webhook=false` and `--enable-mutating-webhook=false` flag respectively.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.4/hack/deploy/stash.sh \
    | bash -s -- --enable-validating-webhook=false --enable-mutating-webhook=false [--rbac]
```

## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](https://github.com/appscode/stash/tree/0.7.0-rc.4/chart/stash) from [AppsCode Charts Repository](https://github.com/appscode/charts). To install the chart with the release name `my-release`:

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm search appscode/stash
NAME            CHART VERSION APP VERSION DESCRIPTION
appscode/stash  0.7.0-rc.4    0.7.0-rc.4  Stash by AppsCode - Backup your Kubernetes Volumes

# Kubernetes 1.8.x
$ helm install appscode/stash --name stash-operator --version 0.7.0-rc.4

# Kubernetes 1.9.0 or later
$ helm install appscode/stash --name stash-operator --version 0.7.0-rc.4 \
  --set apiserver.ca="$(onessl get kube-ca)" \
  --set apiserver.enableValidatingWebhook=true \
  --set apiserver.enableMutatingWebhook=true
```

To install `onessl`, run the following commands:

```console
# Mac OSX amd64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-darwin-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/

# Linux amd64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/

# Linux arm64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-arm64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/
```

To see the detailed configuration options, visit [here](https://github.com/appscode/stash/tree/master/chart/stash).

### Installing in GKE Cluster

If you are installing Stash on a GKE cluster, you will need cluster admin permissions to install Stash operator. Run the following command to grant admin permision to the cluster.

```console
# get current google identity
$ gcloud info | grep Account
Account: [user@example.org]

$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=user@example.org
```


## Verify installation
To check if Stash operator pods have started, run the following command:
```console
$ kubectl get pods --all-namespaces -l app=stash --watch

NAMESPACE     NAME                              READY     STATUS    RESTARTS   AGE
kube-system   stash-operator-859d6bdb56-m9br5   2/2       Running   2          5s
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

Now, to confirm CRD groups have been registered by the operator, run the following command:
```console
$ kubectl get crd -l app=stash

NAME                                 AGE
recoveries.stash.appscode.com        5s
repositories.stash.appscode.com      5s
restics.stash.appscode.com           5s
```

Now, you are ready to [take your first backup](/docs/guides/README.md) using Stash.


## Configuring RBAC
Stash creates multiple CRDs: `Restic`, `Repository` and `Recovery`. Stash installer will create 2 user facing cluster roles:

| ClusterRole         | Aggregates To | Desription                            |
|---------------------|---------------|---------------------------------------|
| appscode:stash:edit | admin, edit   | Allows edit access to Stash CRDs, intended to be granted within a namespace using a RoleBinding. |
| appscode:stash:view | view           | Allows read-only access to Stash CRDs, intended to be granted within a namespace using a RoleBinding. |

These user facing roles supports [ClusterRole Aggregation](https://kubernetes.io/docs/admin/authorization/rbac/#aggregated-clusterroles) feature in Kubernetes 1.9 or later clusters.


## Using kubectl for Restic
```console
# List all Restic objects
$ kubectl get restic --all-namespaces

# List Restic objects for a namespace
$ kubectl get restic -n <namespace>

# Get Restic YAML
$ kubectl get restic -n <namespace> <name> -o yaml

# Describe Restic. Very useful to debug problems.
$ kubectl describe restic -n <namespace> <name>
```


## Using kubectl for Recovery
```console
# List all Recovery objects
$ kubectl get recovery --all-namespaces

# List Recovery objects for a namespace
$ kubectl get recovery -n <namespace>

# Get Recovery YAML
$ kubectl get recovery -n <namespace> <name> -o yaml

# Describe Recovery. Very useful to debug problems.
$ kubectl describe recovery -n <namespace> <name>
```


## Detect Stash version
To detect Stash version, exec into the operator pod and run `stash version` command.

```console
$ POD_NAMESPACE=kube-system
$ POD_NAME=$(kubectl get pods -n $POD_NAMESPACE -l app=stash -o jsonpath={.items[0].metadata.name})
$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE stash version

Version = 0.7.0-rc.4
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = 85b0f16ab1b915633e968aac0ee23f877808ef49
GitBranch = release-0.5
GitTag = 0.7.0-rc.4
CommitTimestamp = 2017-10-10T05:24:23

$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE restic version
restic 0.8.3
compiled with go1.9 on linux/amd64
```

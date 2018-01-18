---
title: Install
description: Stash Install
menu:
  product_stash_0.6.3:
    identifier: install-stash
    name: Install
    parent: setup
    weight: 10
product_name: stash
menu_name: product_stash_0.6.3
section_menu_id: setup
---

# Installation Guide

## Using YAML
Stash can be installed using YAML files includes in the [/hack/deploy](https://github.com/appscode/stash/tree/master/hack/deploy) folder.

```console
# Install without RBAC roles
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/without-rbac.yaml


# Install with RBAC roles
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/with-rbac.yaml
```

## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](https://github.com/appscode/stash/tree/master/chart/stable/stash) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```bash
$ helm repo update
$ helm install stable/stash --name my-release
```
To see the detailed configuration options, visit [here](https://github.com/appscode/stash/tree/master/chart/stable/stash).


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

NAME                            AGE
recoveries.stash.appscode.com   5s
restics.stash.appscode.com      5s
```

Now, you are ready to [take your first backup](/docs/guides/README.md) using Stash.


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

Version = 0.6.3
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = 85b0f16ab1b915633e968aac0ee23f877808ef49
GitBranch = release-0.5
GitTag = 0.6.3
CommitTimestamp = 2017-10-10T05:24:23

$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE restic version
restic 0.8.1
compiled with go1.9 on linux/amd64
```

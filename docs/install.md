> New to Stash? Please start [here](/docs/tutorial.md).

# Installation Guide

## Using YAML
Stash can be installed using YAML files includes in the [/hack/deploy](/hack/deploy) folder.

```console
# Install without RBAC roles
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.5.1/hack/deploy/without-rbac.yaml


# Install with RBAC roles
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.5.1/hack/deploy/with-rbac.yaml
```

## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stash) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```bash
$ helm repo update
$ helm install stable/stash --name my-release
```
To see the detailed configuration options, visit [here](/chart/stash/README.md).


## Verify installation
To check if Stash operator pods have started, run the following command:
```console
$ kubectl get pods --all-namespaces -l app=stash --watch
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

Now, to confirm CRD groups have been registered by the operator, run the following command:
```console
$ kubectl get crd -l app=stash
```

Now, you are ready to [take your first backup](/docs/tutorial.md) using Stash.


## Using kubectl
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


## Detect Stash version
To detect Stash version, exec into the operator pod and run `stash version` command.

```console
$ POD_NAMESPACE=kube-system
$ POD_NAME=$(kubectl get pods -n $POD_NAMESPACE -l app=stash -o jsonpath={.items[0].metadata.name})
$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE stash version

Version = 0.5.1
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = 85b0f16ab1b915633e968aac0ee23f877808ef49
GitBranch = release-0.5
GitTag = 0.5.1
CommitTimestamp = 2017-10-10T05:24:23

$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE restic version
restic 0.7.3
compiled with go1.9 on linux/amd64
```

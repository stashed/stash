> New to Stash? Please start [here](/docs/tutorial.md).

# Installation Guide

## Using YAML
Stash can be installed using YAML files includes in the [/hack/deploy](/hack/deploy) folder.

```console
# Install without RBAC roles
$ curl https://raw.githubusercontent.com/appscode/stash/0.5.0/hack/deploy/without-rbac.yaml \
  | kubectl apply -f -


# Install with RBAC roles
$ curl https://raw.githubusercontent.com/appscode/stash/0.5.0/hack/deploy/with-rbac.yaml \
  | kubectl apply -f -
```

## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stash) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```bash
$ helm install chart/stash --name my-release
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
```console
$ POD_NAMESPACE=kube-system
$ POD_NAME=$(kubectl get pods -n $POD_NAMESPACE -l app=stash -o jsonpath={.items[0].metadata.name})
$ kubectl exec -it $POD_NAME -c operator -n $POD_NAMESPACE stash version

Version = 0.5.0
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = 85b0f16ab1b915633e968aac0ee23f877808ef49
GitBranch = release-0.5
GitTag = 0.5.0
CommitTimestamp = 2017-10-10T05:24:23
```

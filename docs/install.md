> New to Stash? Please start with [here](/docs/tutorial.md).

# Installation Guide

## Using YAML
Stash can be installed using YAML files includes in the [/hack/deploy](/hack/deploy) folder.

```sh
# Install without RBAC roles
$ curl https://raw.githubusercontent.com/appscode/stash/master/hack/deploy/stash-without-rbac.yaml \
  | kubectl apply -f -

# Install with RBAC roles
$ curl https://raw.githubusercontent.com/appscode/stash/master/hack/deploy/stash-with-rbac.yaml \
  | kubectl apply -f -
```

## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stash) included in this repository. To install the chart with the release name `my-release`:
```bash
$ helm install chart/stash --name my-release
```
To see the detailed configuration options, visit [here](/chart/stash/README.md).


## Verify installation
To check if Stash operator pods have started, run the following command:
```sh
$ kubectl get pods --all-namespaces -l app=stash --watch
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

Now, to confirm TPR groups have been registered by the operator, run the following command:
```sh
$ kubectl get thirdpartyresources restic.stash.appscode.com
```

Now, you are ready to [take your first backup](/docs/tutorial.md) using Stash.

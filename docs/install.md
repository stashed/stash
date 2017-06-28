# Installation
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

Once Controller is *Running* It will create the [required ThirdPartyResources for ingress and certificates](/docs/developer-guide#third-party-resources).
Check the Controller is running or not via `kubectl get pods` there should be a pod nameed `appscode-voyager-xxxxxxxxxx-xxxxx`.
Now Create Your Ingress/Certificated.


## Using Helm
Stash can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stash) included in this repository. To install the chart with the release name `my-release`:
```bash
$ helm install chart/stash --name my-release
```
To see the detailed configuration options, visit [here](chart/stash/README.md).


## Verify installation
To check if Stash operator pods have started, run the following command:
```sh
$ kubectl get pods --all-namespaces -l app=stash --watch
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

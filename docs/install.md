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
Stash can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stash) included in this repository. TO see various configuration options, visit [here](chart/stash/README.md).

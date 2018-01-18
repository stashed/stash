---
title: Uninstall
description: Stash Uninstall
menu:
  product_stash_0.6.3:
    identifier: uninstall-stash
    name: Uninstall
    parent: setup
    weight: 20
product_name: stash
menu_name: product_stash_0.6.3
section_menu_id: setup
---
# Uninstall Stash

Please follow the steps below to uninstall Stash:

- Delete the deployment and service used for Stash operator.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/uninstall.sh | bash

+ kubectl delete deployment -l app=stash -n kube-system
deployment "stash-operator" deleted
+ kubectl delete service -l app=stash -n kube-system
service "stash-operator" deleted
+ kubectl delete secret -l app=stash -n kube-system
No resources found
+ kubectl delete serviceaccount -l app=stash -n kube-system
No resources found
+ kubectl delete clusterrolebindings -l app=stash -n kube-system
No resources found
+ kubectl delete clusterrole -l app=stash -n kube-system
No resources found
+ kubectl delete initializerconfiguration -l app=stash
initializerconfiguration "stash-initializer" deleted
```

- Now, wait several seconds for Stash to stop running. To confirm that Stash operator pod(s) have stopped running, run:

```console
$ kubectl get pods --all-namespaces -l app=stash
```

- To keep a copy of your existing `Restic` objects, run:

```console
kubectl get restic.stash.appscode.com --all-namespaces -o yaml > data.yaml
```

- To delete existing `Restic` objects from all namespaces, run the following command in each namespace one by one.

```
kubectl delete restic.stash.appscode.com --all --cascade=false
```

- Delete the old CRD-registration.

```console
kubectl delete crd -l app=stash
```

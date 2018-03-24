---
title: Uninstall
description: Stash Uninstall
menu:
  product_stash_0.7.0-rc.2:
    identifier: uninstall-stash
    name: Uninstall
    parent: setup
    weight: 20
product_name: stash
menu_name: product_stash_0.7.0-rc.2
section_menu_id: setup
---
# Uninstall Stash

To uninstall Stash operator, run the following command:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0-rc.2/hack/deploy/stash.sh \
    | bash -s -- --uninstall [--namespace=NAMESPACE]

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

The above command will leave the Stash crd objects as-is. If you wish to **nuke** all Stash crd objects, also pass the `--purge` flag. This will keep a copy of Stash crd objects in your current directory.

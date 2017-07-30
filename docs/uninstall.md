> New to Stash? Please start [here](/docs/tutorial.md).

# Uninstall Stash
Please follow the steps below to uninstall Stash:

1. Delete the deployment and service used for Stash operator.
```console
$ ./hack/deploy/uninstall.sh
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
```

2. Now, wait several seconds for Stash to stop running. To confirm that Stash operator pod(s) have stopped running, run:
```console
$ kubectl get pods --all-namespaces -l app=stash
```

3. To keep a copy of your existing `Restic` objects, run:
```console
kubectl get restic.stash.appscode.com --all-namespaces -o yaml > data.yaml
```

4. To delete existing `Restic` objects from all namespaces, run the following command in each namespace one by one.
```
kubectl delete restic.stash.appscode.com --all --cascade=false
```

5. Delete the old TPR-registration.
```console
kubectl delete thirdpartyresource -l app=stash
```

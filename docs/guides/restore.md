> New to Stash? Please start [here](/docs/guides/README.md).

# Restore Backup
This tutorial will show you how to restore a Stash backup. At first, backup a kubernetes workload volume by following the steps [here](/docs/guides/backup.md).

To restore a backup, you need to create a `Recovery` CRD by specifying `Restic`, target workload and volume where backup will be restored.

```console
$ kubectl apply -f ./docs/examples/tutorial/recovery.yaml
recovery "stash-demo" created
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: stash-demo
  namespace: default
spec:
  restic: stash-demo
  workload:
    kind: Deployment
    name: stash-demo
  volumes:
  - name: source-data
    hostPath:
      path: /data/stash-test/restic-restored
```

Here,
 - `spec.restic` specifies the `Restic` name that was used to take backups.
 - `spec.workload` specifies a target workload that was backed up using `Restic`. A single `Restic` backups all types of workloads that matches the label-selector, but you can only restore a specific workload using a `Recovery`.
    - For workload kind `Statefulset`, you need to specify pod [index](https://kubernetes.io/docs/guides/stateful-application/basic-stateful-set/#pods-in-a-statefulset) using `spec.podOrdinal`.
    - For workload kind `Daemonset`, you need to specify node name using `spec.nodeName`.
 - `spec.volumes` indicates an array of volumes where snapshots will be recovered. Here, `name` should be same as the workload volume name that was backed up using `Restic`.

Stash operator watches for `Recovery` objects using Kubernetes api. It collects required snapshot information from the specified `Restic` object. Then it creates a recovery job that performs the recovery guides. On completion, job and associated pods are deleted by stash operator. To verify recovery, we can check the `Recovery` status.

```yaml
$ kubectl get recovery stash-demo -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  clusterName: ""
  creationTimestamp: 2017-12-04T06:27:16Z
  deletionGracePeriodSeconds: null
  deletionTimestamp: null
  generation: 0
  initializers: null
  name: stash-demo
  namespace: default
  resourceVersion: "29671"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/recoveries/stash-demo
  uid: 2bf74432-d8bc-11e7-be92-0800277f19c0
spec:
  restic: stash-demo
  volumes:
  - hostPath:
      path: /data/stash-test/restic-restored
    name: source-data
  workload:
    kind: Deployment
    name: stash-demo
status:
  phase: Succeeded
```

## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
$ kubectl delete recovery stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn about the details of Restic CRD [here](/docs/concepts/restic.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Learn about how to configure Stash operator as workload initializer [here](/docs/guides/initializer.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).
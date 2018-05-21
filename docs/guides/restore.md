---
title: Restore Volumes | Stash
description: Restore Volumes using Stash
menu:
  product_stash_0.7.0-rc.4:
    identifier: restore-stash
    name: Restore Volumes
    parent: guides
    weight: 25
product_name: stash
menu_name: product_stash_0.7.0-rc.4
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Restore Backup
This tutorial will show you how to restore a Stash backup. At first, backup a kubernetes workload volume by following the steps [here](/docs/guides/backup.md).

To restore a backup, you need to create a `Recovery` CRD by specifying `Repository`, `path` and volume where the backup will be restored. Here, is a sample `Recovery` to recover the latest snapshot.

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
  repository:
    name: deployment.stash-demo
    namespace: default
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
```

Here,

- `spec.repository.name` specifies the name of the `Repository` crd that represents respective **restic** repository.
- `spec.repository.namespace` specifies the namespace of `Repository` crd.
- `spec.paths` specifies the file-group paths that were backed up using `Restic`.
- `spec.recoveredVolumes` indicates an array of volumes where snapshots will be recovered. Here, `mountPath` specifies where the volume will be mounted. Note that, `Recovery` recovers data in the same paths from where the backup was taken (specified in `spec.paths`). So, volumes must be mounted on those paths or their parent paths.

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
  repository:
    name: deployment.stash-demo
    namespace: default
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
status:
  phase: Succeeded
```

## Recover a specific snapshot

With the help of [Snapshot](/docs/concepts/crds/snapshot.md) object, stash allows the users to recover a particular snapshot. Now, the users can specify which snapshot to recover. Here, is an example of how to recover a specific snapshot.

First, list the available snapshots,

```console
$ kubectl get snapshots --all-namespaces
NAME                             AGE
deployment.stash-demo-d3050010   4m
deployment.stash-demo-300d7c13   3m
deployment.stash-demo-c24f6d96   2m
deployment.stash-demo-80bcc7e3   1m
deployment.stash-demo-3e79020e   35s
``` 

Now, create a `Recovery` with specifying `Snapshot` name,

```console
$ kubectl apply -f ./docs/examples/tutorial/recovery-specific-snapshot.yaml
recovery "stash-demo" created
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: stash-demo
  namespace: default
spec:
  repository:
    name: deployment.stash-demo
    namespace: default
  snapshot: deployment.stash-demo-d3050010
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
```
## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
$ kubectl delete recovery stash-demo
$ kubectl delete repository deployment.stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).
---
title: Repository Overview
menu:
  product_stash_0.8.0:
    identifier: repository-overview
    name: Repository
    parent: crds
    weight: 15
product_name: stash
menu_name: product_stash_0.8.0
section_menu_id: concepts
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Repository

## What is Repository
A `Repository` is a Kubernetes `CustomResourceDefinition (CRD)`. It provides information of a [restic](https://restic.net/) repository in Kubernetes native way. When [stash](/docs/concepts/what-is-stash/overview.md) sidecar creates a restic repository for backup in the desired backend, it also creates a `Repository` CRD object with relevant information of the repository. This enables a user to view backup status of the workloads very easily.

## Repository CRD structure
A sample `Repository` CRD object for backup a `Deployment` in local backend is shown below,

```yaml
apiVersion: stash.appscode.com/v1alpha1
  kind: Repository
  metadata:
    clusterName: ""
    creationTimestamp: 2018-04-10T05:09:10Z
    generation: 0
    labels:
      restic: stash-demo
      workload-kind: Deployment
      workload-name: stash-demo
    name: deployment.stash-demo
    namespace: default
    resourceVersion: "7515"
    selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/repositories/deployment.stash-demo
    uid: 4dccffa8-3c7d-11e8-9f4d-0800270b3cc5
  spec:
    backend:
      local:
        hostPath:
          path: /data/stash-test/restic-repo
        mountPath: /safe/data
        subPath: deployment/stash-demo
      storageSecretName: local-secret
  status:
    backupCount: 7
    firstBackupTime: 2018-04-10T05:10:11Z
    lastBackupDuration: 3.026137088s
    lastBackupTime: 2018-04-10T05:16:12Z
```

Here, we are going describe some important sections of `Repository` CRD.

## Repository Labels

A `Repository` object maintains some important information using labels. These labels enable a user to filter `Repository` according to `restic`, `workload-kind`, `workload-name`, `node-name` etc. Details of these labels are given below.

| Label name      | Description                                                                                                                                                   |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `restic`        | Name of the `Restic` which is responsible for this `Repository`.                                                                                              |
| `workload-kind` | `Kind` of the workload for which the `Repository` has been created.                                                                                           |
| `workload-name` | `Name` of the workload for which the `Repository` has been created.                                                                                           |
| `pod-name`      | This `label` present when the respective workload is  `StatefulSet`. It represents the pod name of the `StatefulSet` who is responsible for this `Repository`. |
| `node-name`     | This `label` present when the respective workload is  `DaemonSet`. It represents the node name where the  `DaemonSet` is running.                              |

## Repository Spec

`Repository` CRD needs the following information in `.spec` section.

### spec.backend

`spec.backend` holds the backend information where the backup snapshots are being stored. To learn how to configure various backends for Restic, please visit [here](/docs/guides/backends/overview.md).

### backend prefix/subPath

`prefix` of any backend denotes the directory inside the backend where the snapshots are being stored. If the backend is a **Local** backend then `subPath` is used for this purpose.

### spec.wipeOut

`spec.wipeOut` field indicates whether stash will delete respective **restic** repository from the backend or only delete `Repository` crd when `Repository` crd is deleted. The default value of this field is `false` which indicates that only `Repository` crd will be deleted from Kubernetes. To know how to use this field to delete restic repository see [here](/docs/concepts/crds/repository.md#delete-respective-restic-repository).

## Repository Status

Stash operator updates `.status` of a Repository CRD every time a backup operation is completed.

- `status.backupCount` indicates the total number of backup operation completed for this Repository.
- `status.firstBackupTime` indicates the timestamp of first backup operation.
- `status.lastBackupTime` indicates the timestamp of last backup operation.
- `status.lastSuccessfulBackupTime` indicates the timestamp of last successful backup operation. If `status.lastBackupTime` and `status.lastSuccessfulBackupTime` are same, it means that last backup operation was successful.
- `status.lastBackupDuration` indicates the duration of last backup operation.

## Creation of Repository CRD

Whenever a `restic` repository is created according to [these](/docs/concepts/crds/restic.md#backup-repository-structure) rules, it also create respective `Repository` CRD object. Name of this `Repository` CRD object is generated based on rules below:

- For workload kind `Deployment`, `Replicaset` and `ReplicationController` `Repository` is created with name `<WORKLOAD_KIND>.<WORKLOAD_NAME>`. For multiple replicas, only one `Repository` is created as the backup is taken by sidecar of the replica determined by leader-election.
- For workload kind `Statefulset`, `Repository` is created with name`<WORKLOAD_KIND>.<POD_NAME>`. A separate `Repository` is created for each replica of a StatefulSet.
- For workload kind `DaemonSet`, Repository is created with name `<WORKLOAD_KIND>.<WORKLOAD_NAME>.<NODE_NAME>`. One repository is created for each node where pods of a DaemonSet are running.

## Working with Repository CRD

Here are some helpful commands to play with `Repository` CRD,

```
# List all Repositories of all namespaces in the cluster
$ kubectl get repository --all-namespaces

# List all Repositories created for a particular Restic
$ kubectl get repository -l restic=stash-demo --all-namespaces

# List all Repositories of Deployment workloads of all namespaces
$ kubectl get repository -l workload-kind=Deployment --all-namespaces

# List all Respositories of a particular Workload
$ kubectl get repository -l workload-kind=StatefulSet,workload-name=stash-demo

# List all Repositories of a particular node (DaemonSet only)
$ kubectl get repository -l node-name=minikube
```

## Deleting Repository

Stash allows the users to delete **only `Repository` crd** or **`Repository` crd with respective restic repository**. Here, we are going to show how to perform these delete operations.

### Delete only Repository crd

 You can delete only `Repository` crd by,

```console
$ kubectl delete repository <repository-name>

# Example
$ kubectl delete repository deployment.stash-demo
repository "deployment.stash-demo" deleted
```

This will delete only `Repository` crd. It won't delete any backed up data from respective restic repository.

>If you delete `Repository` crd while stash-sidecar still exist on the workload, Stash will re-create the `Repository` crd and continue to take backup. In this case, `status` field of `Repository` crd will be reset.

> If you don't want stash to re-create `Repository` crd, you have to stop Stash from taking backup. To see how to stop stash from taking backup see [here](/docs/guides/backup.md#disable-backup).

### Delete respective restic repository

In order to prevent the users from accidentally deleting **restic** repository, stash uses a special `wipeOut` flag in `spec` of `Repository` crd. By default, this flag is set to `wipeOut: false`. If you want to delete respective restic repository while deleting `Repository` crd, you must set this flag to `wipeOut: true`.

> Currently stash supports deleting restic repository only for AWS S3, GCS, Azure and OpenStack Swift backend. S3 compatible other backends such as Minio and Rook also support repository deletion.

Here, is an example of deleting restic repository from Minio backend.

First, set `wipeOut: true` by patching `Repository` crd.

```console
$ kubectl patch repository deployment.stash-demo --type="merge" --patch='{"spec": {"wipeOut": true}}'
repository "deployment.stash-demo" patched
```

Check the repository has been successfully patched correctly.

```console
$ kubectl get repository deployment.stash-demo -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  clusterName: ""
  creationTimestamp: 2018-04-19T06:47:13Z
  finalizers:
  - wipeOut-repository
  generation: 0
  labels:
    restic: minio-restic
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo
  namespace: default
  resourceVersion: "7721"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/repositories/deployment.stash-demo
  uid: 7dd5065e-439d-11e8-8994-080027a9112c
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: http://minio-service.default.svc
      prefix: stash-qa/demo/deployment/stash-demo
    storageSecretName: minio-restic-secret
  wipeOut: true
status:
  backupCount: 1
  firstBackupTime: 2018-04-19T06:47:13Z
  lastBackupDuration: 3.888107169s
  lastBackupTime: 2018-04-19T06:47:13Z
```

Notice that `spec.wipeOut` field is `true`. So, you are ready to delete restic repository. Now, delete `Repository` crd.

```console
$ kubectl delete repository deployment.stash-demo
repository "deployment.stash-demo" deleted
```

If everything goes well, respective restic repository will be deleted from the backend.

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Snapshot [here](/docs/concepts/crds/snapshot.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

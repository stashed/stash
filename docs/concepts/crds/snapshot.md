---
title: Snapshot Overview
menu:
  product_stash_0.8.3:
    identifier: snapshot-overview
    name: Snapshot
    parent: crds
    weight: 50
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---
> New to Stash? Please start [here](/docs/concepts/README.md).

# Snapshot

## What is Snapshot

A `Snapshot` is a representation of backup snapshot in a Kubernetes native way. Stash uses an [Aggregated API Server](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/aggregated-api-servers.md) to provide `get` and `list` capabilities for snapshots from the backend.

This enables you to view some useful information such as `creationTimestamp`, `snapshot id`, `backed up path` etc of a snapshot. This also provides the capability to restore a specific snapshot.

## Snapshot structure

Like other official Kuberentes resources, a `Snapshot` has `TypeMeta`, `ObjectMeta` and `Status` sections. However, unlike other Kubernetes resources, it does not have a `Spec` section.

A sample `Snapshot` object is shown below,

```yaml
apiVersion: repositories.stash.appscode.com/v1alpha1
kind: Snapshot
metadata:
  creationTimestamp: "2019-04-25T05:05:06Z"
  labels:
    repository: local-repo
  name: local-repo-d421fe22
  namespace: demo
  selfLink: /apis/repositories.stash.appscode.com/v1alpha1/namespaces/demo/snapshots/local-repo-d421fe22
  uid: d421fe22de090511a74b8ab5f1f307f2fa4e0d8e2f624a7481095db828127147
status:
  gid: 0
  hostname: host-0
  paths:
  - /source/data
  tree: bb1d63756e937c001cf48ed062c69a9968978821a328f5ab06873f3b90346da2
  uid: 0
  username: ""
```

Here, we are going to describe the various sections of a `Snapshot` object.

### Snapshot `Metadata`

- **metadata.name**

  `metadata.name` specifies the name of the `Snapshot` object. It follows the following pattern, `<Repository crd name>-<first 8 digits of snapshot id>`.

- **metadata.uid**

  `metadata.uid` specifies the complete id of the respective restic snapshot in the backend.

- **metadata.creationTimestamp**

  `metadata.creationTimestamp` represents the time when the snapshot was created.

- **metadata.labels**

  A `Snapshot` object holds `Repository` name as a label in `metadata.labels` section. This helps a user to query the snapshots of a particular `Repository`.

### Snapshot `Status`

`Snapshot` object has the following fields in `.status` section:

- **status.gid**
`status.gid` indicates the group identifier of the user who took this backup.

- **status.hostname**
`status.hostname` indicates the name of the **host** whose data was backed up in this snapshot. For `Deployment`,`ReplicaSet` and `ReplicationController` it is **host-0**. For `DaemonSet`, hostname is the respective **node name** where daemon pod is running. For `StatefulSet`, hostname is **host-\<pod ordinal\>** for the respective pods.

- **status.path**
`status.path` indicates the path that has been backed up in this snapshot.

- **status.tree**
`status.tree` indicates `tree` of the restic snapshot. For more details, please visit [here](https://restic.readthedocs.io/en/stable/100_references.html#trees-and-data).

- **status.uid**
`status.uid` indicates `uid` of the user who took this backup. For `root` user it is 0.

- **status.username**
`status.username` indicates the name of the user who runs the backup process that took the backup.

- **status.tags**
`status.tags` indicates the tags of the snapshot.

## Working with Snapshot

**Listing Snapshots:**

```console
# List Snapshots of all Repositories in the current namespace
$ kubectl get snapshot

# List Snapshots of all Repositories of all namespaces
$ kubectl get snapshot --all-namespaces

# List Snapshots of all Repositories of a particular namespace
$ kubectl get snapshot -n demo

# List Snapshots of a particular Repository
$ kubectl get snapshot -l repository=local-repo

# List Snapshots from multiple Repositories
$ kubectl get snapshot -l 'repository in (local-repo,gcs-repo)'
```

**Viewing information of a particular Snapshot:**

```console
$ kubectl get snapshot [-n <namespace>] <snapshot name> -o yaml

# Example:
$ kubectl get snapshot -n demo local-repo-02b0ed42 -o yaml
```

```yaml
apiVersion: repositories.stash.appscode.com/v1alpha1
kind: Snapshot
metadata:
  creationTimestamp: "2019-04-25T06:01:04Z"
  labels:
    repository: local-repo
  name: local-repo-02b0ed42
  namespace: demo
  selfLink: /apis/repositories.stash.appscode.com/v1alpha1/namespaces/demo/snapshots/local-repo-02b0ed42
  uid: 02b0ed42791d2d13756cb9e2d05db42f514d51e23028f42bcbe3a152978aa499
status:
  gid: 0
  hostname: host-0
  paths:
  - /source/data
  tree: bb1d63756e937c001cf48ed062c69a9968978821a328f5ab06873f3b90346da2
  uid: 0
  username: ""
```

## Preconditions for Snapshot

1. Stash provides `Snapshots` listing facility with the help of an Aggregated API Server. Your cluster must support Aggregated API Server. Otherwise, you won't be able to perform `get` or `list` operation on `Snapshot`.

2. If you are using [local](/docs/guides/backends/local.md) backend, the respective pod that took the backup must be in `Running` state. It is not necessary if you use cloud backends.

## Next Steps

- Learn how to configure `BackupConfiguration` to backup workloads data from [here](/docs/guides/latest/workloads/backup.md).
- Learn how to configure `BackupConfiguration` to backup databases from [here](/docs/guides/latest/databases/backup.md).
- Learn how to configure `BackupConfiguration` to backup stand-alone PVC from [here](/docs/guides/latest/volumes/backup.md).

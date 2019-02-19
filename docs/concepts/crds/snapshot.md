---
title: Snapshot Overview
menu:
  product_stash_0.8.3:
    identifier: snapshot-overview
    name: Snapshot
    parent: crds
    weight: 25
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---
> New to Stash? Please start [here](/docs/concepts/README.md).

# Snapshot

## What is Snapshot
A `Snapshot` is a representation of [restic](https://restic.net/) backup snapshot in a Kubernetes native way. With the help of [Aggregated API Servers](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/aggregated-api-servers.md), **Stash** provides the users a way to `view`,`list` and `delete` snapshots from restic repositories. Now, a user can view some crucial information of snapshots like `creationTimestamp`, `snapshot id`, `backed up path`  etc. In future, this will enable the users to recover a particular snapshot using stash.

## Snapshot structure
A sample `Snapshot` object's structure created by backing up a `Deployment` is shown below,

```yaml
apiVersion: repositories.stash.appscode.com/v1alpha1
kind: Snapshot
metadata:
  creationTimestamp: 2018-04-09T06:43:03Z
  labels:
    repository: deployment.stash-demo
    restic: stash-demo
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo-11156792
  namespace: default
  selfLink: /apis/repositories.stash.appscode.com/v1alpha1/namespaces/default/snapshots/deployment.stash-demo-11156792
  uid: 11156792ffe5a52ef076c4e7c74f79a4e6ad6f8d4d2a8a078cf9ee507a8f360c
status:
  gid: 0
  hostname: stash-demo
  paths:
  - /source/data
  tree: ab2311afd593e5ef6f95df652215c9d1102b1731b72e2784386350ae95f1a145
  uid: 0
  username: ""

```

Here, we are going to describe some important sections of `Snapshot` object.

### Snapshot Name

`name` filed in `metadata` of a `Snapshot` object represent its name. It follows this pattern,
`<respective repository crd name>-<first 8 digits of snapshot id>`

### Snapshot UID

`uid` field in `metadata` of `Snapshot` object represents complete restic snapshot id.

### CreationTimestamp
`creationTimestamp` in `metadata` field of a `Snapshot` object represents the time when the snapshot was created.

### Snapshot Labels

A `Snapshot` object maintains some important information using labels. These labels enable a user to filter `Snapshot` according to `repository`, `restic`, `workload-kind`, `workload-name`, `node-name` etc. Details of these labels are given below.

| Label name      | Description                                                                                                                                                 |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `repository`    | Name of the `Repository`  where this `Snapshot` is stored.                                                                                                  |
| `restic`        | Name of the `Restic` which is responsible for this `Snapshot`.                                                                                              |
| `workload-kind` | `Kind` of the workload for which the `Snapshot` has been created.                                                                                           |
| `workload-name` | `Name` of the workload for which the `Snapshot` has been created.                                                                                           |
| `pod-name`      | This `label` present when the respective workload is `StatefulSet`. It represents the pod name of the `StatefulSet` who is responsible for this `Snapshot`. |
| `node-name`     | This `label` present when the respective workload is `DaemonSet`. It represents the node name where the `DaemonSet` is running.                             |

### Snapshot Status

`Snapshot` object has following status fields,

* `status.gid` indicates the group identifier of the user who took this backup.
* `status.hostname` indicates the name of the host object whose data is backed up in this snapshot. For `Deployment`,`ReplicaSet` and `ReplicationController` it is workload name. For `DaemonSet` hostname is node name and for `StatefulSet` hostname is pod name.
* `status.path` indicates the path that is backed up in this snapshot.
* `status.tree` indicates `tree` of the restic snapshot. For more details see [here](https://restic.readthedocs.io/en/stable/100_references.html#trees-and-data).
* `status.uid` indicates id of the user who took this backup. For `root` user it is 0.
* `status.username` indicates the name of the user.
* `status.tags` indicates tags of the snapshot.

## Working with Snapshot

**Listing Snapshots:**

```console
# List Snapshots of all repositories in the current namespace
$ kubectl get snapshot

# List Snapshots of a particular repository
$ kubectl get snapshot -l repository=deployment.stash-demo

# List all Snapshots of a particular workload type
$ kubectl get snapshot -l workload-kind=Deployment

# List Snapshots of a particular workload
$ kubectl get snapshot -l workload-name=stash-demo

# List all Snapshots created by a particular restic
$ kubectl get snapshot -l restic=stash-demo

# List Snapshots of a particular pod(only for StatefulSet)
$ kubectl get snapshot -l pod-name=stash-demo-0

# List Snapshots of a particular node(only for DaemonSet)
kubectl get snapshot -l node-name=minikube

# List Snapshot of specific repositories
$ kubectl get snapshot -l 'repository in (deployment.stash-demo,statefulset.stash-demo-0)'
```

**Viewing information of a particular Snapshot:**

```console
$ kubectl get snapshot <snapshot name> -o yaml

# Example:
$ kubectl get snapshot deployment.stash-demo-3d8cd994 -o yaml
```

```yaml
apiVersion: repositories.stash.appscode.com/v1alpha1
kind: Snapshot
metadata:
  creationTimestamp: 2018-04-09T10:06:04Z
  labels:
    repository: deployment.stash-demo
    restic: stash-demo
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo-3d8cd994
  namespace: default
  selfLink: /apis/repositories.stash.appscode.com/v1alpha1/namespaces/default/snapshots/deployment.stash-demo-3d8cd994
  uid: 3d8cd994a3bddcff1d16ac166601d93454d0eb31e2240801e7c1ec030e5af0bf
status:
  gid: 0
  hostname: stash-demo
  paths:
  - /source/data
  tree: ab2311afd593e5ef6f95df652215c9d1102b1731b72e2784386350ae95f1a145
  uid: 0
  username: ""
```

**Deleting a particular Snapshot:**

```console
$ kubectl delete snapshot <snapshot name>

# Example:
$ kubectl delete snapshot statefulset.stash-demo-0-d690726d
snapshot "statefulset.stash-demo-0-d690726d" deleted

```

## Precondition for Snaphsot

1. Stash provides `Snapshots` listing facility with the help of aggregated api server. Stash start aggregated api server if any of the `ValidatingWebhook` and `MutatingWebhook` is enabled. If both of the webhooks are disabled or if your cluster does not support aggregated api server or webhooks, you won't able to list `Snapshot`.
2. If you are using `hostPath` for `Restic` backend, stash takes help of workload pod to provide snapshot list. In this case, workload pod must be running while listing `Snapshot`.
3. If you are using [offline backup](/docs/guides/offline_backup.md) and `hostPath` as your `Restic` backend, you won't able to list `Snapshot`.

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

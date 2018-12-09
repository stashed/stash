---
title: Restic Overview
menu:
  product_stash_0.8.1:
    identifier: restic-overview
    name: Restic
    parent: crds
    weight: 10
product_name: stash
menu_name: product_stash_0.8.1
section_menu_id: concepts
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Restics

## What is Restic
A `Restic` is a Kubernetes `CustomResourceDefinition` (CRD). It provides declarative configuration for [restic](https://restic.net) in a Kubernetes native way. You only need to describe the desired backup operations in a Restic object, and the Stash operator will reconfigure the matching workloads to the desired state for you.

## Restic Spec
As with all other Kubernetes objects, a Restic needs `apiVersion`, `kind`, and `metadata` fields. It also needs a `.spec` section. Below is an example Restic object.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: stash-demo
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

The `.spec` section has following parts:

### spec.selector
`spec.selector` is a required field that specifies a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the Deployments, ReplicaSets, ReplicationControllers, DaemonSets and StatefulSets targeted by this Restic. Selectors are always matched against the labels of Deployments, ReplicaSets, ReplicationControllers, DaemonSets and StatefulSets in the same namespace as Restic object itself. You can create Deployment, etc and its matching Restic is any order. As long as the labels match, Stash operator will add sidecar container to the workload.  If multiple `Restic` objects are matched to a given workload, Stash operator will error out and avoid adding sidecar container.

### spec.type
The default value for `spec.type` is `online`. For offline backup you need to specify `spec.type=offline`. For more details see [here](/docs/guides/offline_backup.md).

### spec.fileGroups
`spec.fileGroups` is a required field that specifies one or more directories that are backed up by [restic](https://restic.net). For each directory, you can specify custom tags and retention policy for snapshots.

 - `spec.fileGroups[].path` represents a local directory that backed up by `restic`.
 - `spec.fileGroups[].tags` is an optional field. This can be used to apply one or more custom tag to snapshots taken from this path.
 - `spec.fileGroups[].retentionPolicyName` is an optional field that is used to specify a retention policy defined in `spec.retentionPolicies`. This defines how old snapshots are forgot by `restic`. If set, these options directly translate into flags for `restic forget` command.

### spec.retentionPolicies

`spec.retentionPolicies` defines an array of retention policies for old snapshots. Retention policy options are below.

| Policy        | Value   | restic forget flag | Description                                                                                        |
|---------------|---------|--------------------|----------------------------------------------------------------------------------------------------|
| `name`        | string  |                    | Name of retention policy provided by users. This is used in file groups to refer to a policy.       |
| `keepLast`    | integer | --keep-last n      | Never delete the n last (most recent) snapshots                                                    |
| `keepHourly`  | integer | --keep-hourly n    | For the last n hours in which a snapshot was made, keep only the last snapshot for each hour.      |
| `keepDaily`   | integer | --keep-daily n     | For the last n days which have one or more snapshots, only keep the last one for that day.         |
| `keepWeekly`  | integer | --keep-weekly n    | For the last n weeks which have one or more snapshots, only keep the last one for that week.       |
| `keepMonthly` | integer | --keep-monthly n   | For the last n months which have one or more snapshots, only keep the last one for that month.     |
| `keepYearly`  | integer | --keep-yearly n    | For the last n years which have one or more snapshots, only keep the last one for that year.       |
| `keepTags`    | array   | --keep-tag <tag>   | Keep all snapshots which have all tags specified by this option (can be specified multiple times). [`--tag foo,tag bar`](https://github.com/restic/restic/blob/master/doc/060_forget.rst) style tagging is not supported. |
| `prune`       | bool    | --prune            | If set, actually removes the data that was referenced by the snapshot from the repository.         |
| `dryRun`      | bool    | --dry-run          | Instructs `restic` to not remove anything but print which snapshots would be removed.              |

You can set one or more of these retention policy options together. To learn more, read [here](
https://restic.readthedocs.io/en/latest/manual.html#removing-snapshots-according-to-a-policy).

### spec.backend
To learn how to configure various backends for Restic, please visit [here](/docs/guides/backends.md).

### spec.schedule
`spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates how often `restic` commands are invoked for file groups.
At each tick, `restic backup` and `restic forget` commands are run for each of the configured file groups.

### spec.paused
`spec.paused` can be used as `enable/disable` switch for Restic. The default value is `false`. To stop restic from taking backup set `spec.paused: true`. For more details see [here](/docs/guides/backup.md#disable-backup).

### spec.resources
`spec.resources` refers to compute resources required by the `stash` sidecar container. To learn more, visit [here](http://kubernetes.io/docs/user-guide/compute-resources/).

### spec.volumeMounts
`spec.volumeMounts` refers to volumes to be mounted in `stash` sidecar to get access to fileGroup paths.

## Backup Repository Structure

 - For workload kind `Deployment`, `ReplicaSet` and `ReplicationController` restic repo is created in the sub-directory `<WORKLOAD_KIND>/<WORKLOAD_NAME>`. For multiple replicas, only one repository is created and sidecar is added to only one pod selected by leader-election.
 - For workload kind `Statefulset` restic repository is created in the sub-directory `<WORKLOAD_KIND>/<POD_NAME>`. For multiple replicas, multiple repositories are created and sidecar is added to all pods.
 - For workload kind `DaemonSet` restic repository is created in the sub-directory `<WORKLOAD_KIND>/<WORKLOAD_NAME>/<NODE_NAME>`. Separate repositories are created for each node and sidecar is added to all pods.

## Prefix for Repository Directory

Stash allow the users to provide a prefix for the backup repository directory. You can provide the prefix using  `local.subPath` for [local bckend](/docs/guides/backends.md#local) and `<backend-type>.prefix` for [other backends](/docs/guides/backends.md#aws-s3) in `spec.backend` field of `Restic` crd.

If you provide the prefix then the repository will be created in the following directory,

1. `<PREFIX>/<WORKLOAD_KIND>/<WORKLOAD_NAME>` for Deployment, ReplicaSet, ReplicationController.
2. `<PREFIX>/<WORKLOAD_KIND>/<POD_NAME>` for StatefulSets.
3. `<PREFIX>/<WORKLOAD_KIND>/<WORKLOAD_NAME>/<NODE_NAME>` for DaemonSets.

This prefix is particularly helpful when you are using a single bucket to backup your workload in the following scenario,

1. You have two or more workload with the same name but in different namespaces.
2. You have two or more cluster in different regions with the exact set-up. i.e. workload names are same in all cluster.

In these scenarios, `Restic` crd without a prefix will cause conflict in the repository directory in the bucket. You can overcome it using a different prefix for each workload.

## Workload Annotations
For each workload where a sidecar container is added by Stash operator, the following annotations are added:

 - `restic.appscode.com/last-applied-configuration` indicates the configuration of applied Restic CRD.
 - `restic.appscode.com/tag` indicates the tag of `appscode/stash` Docker image that was added as a sidecar.

## Updating Restic
The sidecar container watches for changes in the Restic fileGroups, backend and schedule. These changes are automatically applied on the next run of `restic` commands. If the selector of a Restic CRD is changed, Stash operator will update workload accordingly by adding/removing sidecars as required.

## Disable Restic
To stop Restic from taking backup, you can do the following things:

* Set `spec.paused: true` in Restic `yaml` and then update the Restic object. This means:

  - Paused Restic CRDs will not be applied to newly created workloads.
  - Stash sidecar containers will not be removed from existing workloads but the sidecar will stop taking backup.

* Delete the Restic CRD. Stash operator will remove the sidecar container from all matching workloads.

* Change the labels of a workload. Stash operator will remove the sidecar container from that workload. This way you can selectively stop backup of a Deployment, ReplicaSet etc.

For more details about how to disable and resume Restic see [here](/docs/guides/backup.md#disable-backup).

## Next Steps

- Learn about Repository CRD [here](/docs/concepts/crds/repository.md)
- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

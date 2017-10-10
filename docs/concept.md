> New to Stash? Please start [here](/docs/tutorial.md).

# Restics

## What is Restic
A `Restic` is a Kubernetes `CustomResourceDefinition` (CRD). It provides declarative configuration for [restic](https://github.com/restic/restic) in a Kubernetes native way. You only need to describe the desired backup operations in a Restic object, and the Stash operator will reconfigure the matching workloads to the desired state for you.

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
    retentionPolicy:
      keepLast: 5
      prune: true
  backend:
    local:
      path: /safe/data
      volumeSource:
        emptyDir: {}
    storageSecretName: stash-demo
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```

The `.spec` section has 4 main parts:

### .spec.selector
`.spec.selector` is a required field that specifies a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the Deployments, ReplicaSets, ReplicatinControllers, DaemonSets and StatefulSets targeted by this Restic. Selectors are always matched against the labels of Deployments, ReplicaSets, ReplicatinControllers, DaemonSets and StatefulSets in the same namespace as Restic object itself. You can create Deployment, etc and its matching Restic is any order. As long as the labels match, Stash operator will add sidecar container to the workload.  If multiple `Restic` objects are matched to a given workload, Stash operator will error out and avoid adding sidecar container.

### spec.fileGroups
`spec.fileGroups` is a required field that specifies one or more directories that are backed up by [restic](https://github.com/restic/restic). For each directory, you can specify custom tags and retention policy for snapshots.

 - `spec.fileGroups[].path` represents a local directory that backed up by `restic`.
 - `spec.fileGroups[].tags` is an optional field. This can be used to apply one or more custom tag to snapshots taken from this path.
 - `spec.fileGroups[].retentionPolicy` is an optional field. This defines how old snapshots are forgot by `restic`. If set, these options directly translate into flags for `restic forget` command. Retention policy options are below.

| Policy        | Value   | restic forget flag | Description                                                                                        |
|---------------|---------|--------------------|----------------------------------------------------------------------------------------------------|
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
To learn how to configure various backends for Restic, please visit [here](/docs/backends.md).

### spec.schedule
`spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates how often `restic` commands are invoked for file groups.
At each tick, `restic backup` and `restic forget` commands are run for each of the configured file groups.

### spec.useAutoPrefix
When workloads use more than one replicas, the `restic` repository path needs to be set so that data from different replicas do not overwrite one another. `spec.useAutoPrefix` defines how Stash modifies backend repository prefix to handle this. There are 4 possible options.

 - `Smart` option modifies repository prefix based on the workload kind. _This is the default value. This option is used, when no value is set. Usually, you should not need to use any other options._ This is how it works:
    - StatefulSet: Adds Pod name as prefix to user provided backend prefix. If your StatefulSet dynamically allocates PVCs, this helps to backup them in their own `restic` repository.
    - DaemonSet: Adds Node name as prefix to user provided backend prefix. This allows you to backup data from each node on a separate `restic` repository.
    - Deployment, ReplicaSet, ReplicationController: Uses user provided backend prefix unchanged.
 - `NodeName` option adds Node name to backend prefix for any type of workload.
 - `PodName` option adds Pod name to backend prefix for any type of workload.
 - `None` option uses user provided backend prefix unchanged for any type of workload.

### spec.resources
`spec.resources` refers to compute resources required by the `stash` sidecar container. To learn more, visit [here](http://kubernetes.io/docs/user-guide/compute-resources/).

### spec.volumeMounts
`spec.volumeMounts` refers to volumes to be mounted in `stash` sidecar to get access to fileGroup paths.

## Restic Status
Stash operator updates `.status` of a Restic tpr every time a backup operation is completed. 

 - `status.backupCount` indicated the total number of backup operation completed for this Restic tpr.
 - `status.firstBackupTime` indicates the timestamp of first backup operation.
 - `status.lastBackupTime` indicates the timestamp of last backup operation.
 - `status.lastSuccessfulBackupTime` indicates the timestamp of last successful backup operation. If `status.lastBackupTime` and `status.lastSuccessfulBackupTime` are same, it means that last backup operation was successful.
 - `status.lastBackupDuration` indicates the duration of last backup operation.

## Workload Annotations
For each workload where a sidecar container is added by Stash operator, the following annotations are added:
 - `restic.appscode.com/config` indicates the name of Restic tpr.
 - `restic.appscode.com/tag` indicates the tag of `appscode/stash` Docker image that was added as sidecar.

## Updating Restic
The sidecar container watches for changes in the Restic fileGroups, backend and schedule. These changes are automatically applied on the next run of `restic` commands. If the selector of a Restic tpr
is changed, Stash operator will update workload accordingly by adding/removing sidecars as required.

## Disable Backup
To stop taking backup, you can do 2 things:

- Delete the Restic tpr. Stash operator will remove the sidecar container from all matching workloads.
- Change the labels of a workload. Stash operator will remove sidecar container from that workload. This way you can selectively stop backup of a Deployment, ReplicaSet, etc.

## Restore Backup
No special support is required to restore backups taken via Stash. Just run the standard `restic restore` command to restore files from backends. To learn more please visit [here](https://restic.readthedocs.io/en/latest/manual.html#restore-a-snapshot).

_NB_: We are gathering ideas on how to improve the UX for recovery process. Please share your ideas/use-cases [here](https://github.com/appscode/stash/issues/131).

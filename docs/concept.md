> New to Stash? Please start with [here](/docs/tutorial.md).

# Restics

## What is Restic
A `Restic` is a Kubernetes `Third Party Object` (TPR). It provides declarative configuration for [restic](https://github.com/restic/restic) in a Kubernetes native way. You only need to describe the desired backup operations in a Restic object, and the Stash operator will reconfigure the matching workloads to the desired state for you.

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
  - path: /lib
    retentionPolicy:
      keepLast: 5
  backend:
    local:
      path: /repo
      volume:
        emptyDir: {}
        name: repo
    repositorySecretName: stash-demo
  schedule: '@every 1m'
```

The `.spec` section has 4 main parts:

### .spec.selector
`.spec.selector` is a required field that specifies a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the Deployments, ReplicaSets, ReplicatinControllers, DaemonSets and StatefulSets targeted by this Restic. Selectors are always matched against the labels of Deployments, ReplicaSets, ReplicatinControllers, DaemonSets and StatefulSets in the same namespace as Restic object itself. You can create Deployment, etc and its matching Restic is any order. As long as the labels match, Stash operator will add sidecar container to the workload.

### spec.fileGroups
`spec.fileGroups` is a required field that specifies one or more directories that are backed up by [restic](https://github.com/restic/restic). For each directory, you can specify custom tags and retention policy for snapshots.

 - `spec.fileGroups[].path` represents a local directory that backed up by `restic`.
 - `spec.fileGroups[].tags` is an optional field. This can be used to apply one or more custom tag to snaphsots taken from this path.
 - `spec.fileGroups[].retentionPolicy` is an optional field. This defines how old snapshots are forgot and pruned by `restic`. If set, these options directly translate into flags for `restic forget` command. Stash always runs `restic forget` command with `--prune` option to actually remove the data that was referenced by the snapshot from the repository. Retention policy options are below.

| Policy        | Value   | restic forget flag | Description                                                                                        |
|---------------|---------|--------------------|----------------------------------------------------------------------------------------------------|
| `keepLast`    | integer | --keep-last n      | Never delete the n last (most recent) snapshots                                                    |
| `keepHourly`  | integer | --keep-hourly n    | For the last n hours in which a snapshot was made, keep only the last snapshot for each hour.      |
| `keepDaily`   | integer | --keep-daily n     | For the last n days which have one or more snapshots, only keep the last one for that day.         |
| `keepWeekly`  | integer | --keep-weekly n    | For the last n weeks which have one or more snapshots, only keep the last one for that week.       |
| `keepMonthly` | integer | --keep-monthly n   | For the last n months which have one or more snapshots, only keep the last one for that month.     |
| `keepYearly`  | integer | --keep-yearly n    | For the last n years which have one or more snapshots, only keep the last one for that year.       |
| `keepTags`    | array   | --keep-tag <tag>   | Keep all snapshots which have all tags specified by this option (can be specified multiple times). |

You can set one or more of these retention policy options together. To learn more, read [here](
https://restic.readthedocs.io/en/latest/manual.html#removing-snapshots-according-to-a-policy).

### spec.backend
To learn how to configure various backends for Restic, please visit [here](/docs/backends.md).

### spec.schedule
`spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates how often `restic` commands are invokved for file groups.
At each tick, `restic backup` and `restic forget` commands are run for each of the configured file groups.

## Restic Status
Stash operator updates `.status` of a Restic tpr everytime a backup operation is completed. 

 - `status.backupCount` indicated the total number of backup operation completed for this Restic tpr.
 - `status.firstBackupTime` indicates the timestamp of first backup operation.
 - `status.lastBackupTime` indicates the timestamp of last backup operation.
 - `status.lastSuccessfulBackupTime` indicates the timestamp of last successful backup operation. If `status.lastBackupTime` and `status.lastSuccessfulBackupTime` are same, it means that last backup operation was successful.
 - `status.lastBackupDuration` indicates the duration of last backup operation.

## Workload Annotations
For each workload where a sidecar container is added by Stash operator, the following annotations are added:
 - `restic.appscode.com/config` indicates the name of Restic tpr.
 - `restic.appscode.com/tag` indicates the tag of appscode/stash Docker image that was added as sidecar.

## Updating Restic
The sidecar container watches for changes in the Restic fileGroups, backend and schedule. These changes are automatically applied on the next run of `restic` commands. If the selector of a Restic tpr
is changed, Stash operator will update workload accordingly by adding/removing sidecars as required.

## Disable Backup
To stop taking backup, you can do 2 things:

- Delete the Restic tpr. Stash operator will remove the sidecar container from all matching workloads.
- Change the labels of a workload. Stash operator will remove sidecar container from that workload. This way you can selectively stop backup of a Deployment, ReplicaSet, etc.

## Restore Backup
No special support is required to restore backups taken via Stash. Just run the standard `restic restore` command to restore files from backends. To learn more please visit [here](https://restic.readthedocs.io/en/latest/manual.html#restore-a-snapshot).

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
      keepLastSnapshots: 5
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

- spec.fileGroups[].path represents a local directory that backed up by `restic`.

- spec.fileGroups[].tags is an optional field. This can be used to apply one or more custom tag to snaphsots taken from this path.

- spec.fileGroups[].retentionPolicy is an optional field. This defines how old snapshots are forgot and pruned by `restic`. If set, these options directly translate into flags for `restic forget` command. Stash always runs `restic forget` command with `--prune` option to actually remove the data that was referenced by the snapshot from the repository. Retention policy options are below.

| Policy                 | Value   | restic forget flag | Description                                                                                        |
|------------------------|---------|--------------------|----------------------------------------------------------------------------------------------------|
| `keepLastSnapshots`    | integer | --keep-last n      | Never delete the n last (most recent) snapshots                                                    |
| `keepHourlySnapshots`  | integer | --keep-hourly n    | For the last n hours in which a snapshot was made, keep only the last snapshot for each hour.      |
| `keepDailySnapshots`   | integer | --keep-daily n     | For the last n days which have one or more snapshots, only keep the last one for that day.         |
| `keepWeeklySnapshots`  | integer | --keep-weekly n    | For the last n weeks which have one or more snapshots, only keep the last one for that week.       |
| `keepMonthlySnapshots` | integer | --keep-monthly n   | For the last n months which have one or more snapshots, only keep the last one for that month.     |
| `keepYearlySnapshots`  | integer | --keep-yearly n    | For the last n years which have one or more snapshots, only keep the last one for that year.       |
| `keepTags`             | array   | --keep-tag <tag>   | Keep all snapshots which have all tags specified by this option (can be specified multiple times). |

You can set one or more of these retention policy options together. To learn more, read [here](
https://restic.readthedocs.io/en/latest/manual.html#removing-snapshots-according-to-a-policy).

### spec.schedule
`spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates how often `restic` commands are invokved for file groups.
At each tick, `restic backup` and `restic forget` commands are run for each of the configured file groups.












Status
Annotation





## Backup Nodes

If one interested in take backup of host paths, this can be done by deploying a `DaemonSet` with a do nothing busybox container. 
Stash TPR controller can use that as a vessel for running restic sidecar containers.

## Update Backup

One can update the source, retention policy, tags, cron schedule of the Stash object. After updating the Stash object backup process will follow the new backup strategy.
If user wants to update the image of restic-sidecar container he/she needs to update the `restic.appscode.com/image` in field annotation in the backup object. This will automatically update the restic-sidecar container.
In case of Statefulset user needs to update the sidecar container manually.

## Disable Backup

For disabling backup process one needs to delete the corresponding Stash object in case of `RC`, `Replica Set`, `Deployment`, `DaemonSet`.
In case of `Statefulset` user needs to delete the corresponding backup object as well as remove the side container from the Statefulset manually.












## Enable Backup

For enabling the backup process for a particular kubernetes object like `RC`, `Replica Set`, `Deployment`, `DaemonSet` user adds a label `restic.appscode.com/config: <name_of_tpr>`. `<name_of_tpr>` is the name of Stash object. And then user creates the Stash object for starting backup process.
In case of StaefulSet user has to add the restic-sidecar container manually.

```yaml
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  labels:
    restic.appscode.com/config: test-backup
  name: test-statefulset
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  serviceName: test-svc
  template:
    metadata:
      labels:
        app: nginx
      name: nginx
    spec:
      containers:
      - image: nginx
        imagePullPolicy: Always
        name: nginx
        volumeMounts:
        - mountPath: /source_path
          name: test-volume
      - args:
        - watch
        - --v=10
        env:
        - name: STASH_NAMESPACE
          value: default
        - name: TPR
          value: test-backup
        image: appscode/stash:latest
        imagePullPolicy: Always
        name: restic-sidecar
        volumeMounts:
        - mountPath: /source_path
          name: test-volume
        - mountPath: /repo_path
          name: stash-vol
      volumes:
      - emptyDir: {}
        name: test-volume
      - emptyDir: {}
        name: stash-vol
```


# Architecture 

This guide will walk you through the architectural design of Backup Controller.

## Backup Controller:
Backup Controller collects all information from watcher. This Watcher watches Backup objects. 
Controller detects following ResourceEventType:

* ADDED
* UPDATETD
* DELETED

## Workflow
User deploys stash TPR controller. This will automatically create TPR if not present.
User creates a TPR object defining the information needed for taking backups. User adds a label `restic.appscode.com/config:<name_of_tpr>` Replication Controllers, Deployments, Replica Sets, Replication Controllers, Statefulsets that TPR controller watches for. 
Once TPR controller finds RC etc that has enabled backup, it will add a sidecar container with stash image. So, stash will restart the pods for the first time. In restic-sidecar container backup process will be done through a cron schedule.
When a snapshot is taken an event will be created under the same namespace. Event name will be `<name_of_tpr>-<backup_count>`. If a backup precess successful event reason will show us `Success` else event reason will be `Failed`
If the RC, Deployments, Replica Sets, Replication Controllers, and TPR association is later removed, TPR controller will also remove the side car container.

## Entrypoint

Since restic process will be run on a schedule, some process will be needed to be running as the entrypoint. 
This is a loader type process that watches restic TPR and translates that into the restic compatiable config. eg,

* stash run: This is the main TPR controller entrypoint that is run as a single deployment in Kubernetes.
* stash watch: This will watch Kubernetes restic TPR and start the cron process.

## Restarting pods

As mentioned before, first time side car containers are added, pods will be restarted by controller. Who performs the restart will be done on a case-by-case basis. 
For example, Kubernetes itself will restarts pods behind a deployment. In such cases, TPR controller will let Kubernetes do that.

## Original Tracking Issue:
https://github.com/appscode/stash/issues/1

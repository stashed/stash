



## High Level Tasks
* Create `restic.stash.appscode.com` Third Party Resource
* Create Backup Deployment

## Deploying Backup

### Create Third Party Resource
`Backup process` depends on Third Party Resource Object `restic.stash.appscode.com`. This object can be created using following data.

```yaml
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
metadata:
  name: restic.stash.appscode.com
description: "Stash by AppsCode - Backup & restore Kubernetes volumes"
versions:
  - name: v1alpha1
```


```sh
# Create Third Party Resource
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/master/api/extensions/backup.yaml
```


### Deploy Controller
Stash controller communicates with kube-apiserver at inCluster mode if no master or kubeconfig is provided. It watches Stash resource to handle backup process.
```
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/master/hack/deploy/deployments.yaml
```

#### Configuration Options
```
--master          // The address of the Kubernetes API server (overrides any value in kubeconfig)
--kubeconfig      // Path to kubeconfig file with authorization information (the master location is set by the master flag)
--image           // Stash image name with version to be run in restic-sidecar (appscode/stash:latest)
```

## Stash
This resource type is backed by a controller which take backup of kubernetes volumes from any running pod in Kubernetes. It can also take backup of host paths from Nodes in Kubernetes.

### Resource
A AppsCode Stash resource Looks like at the kubernetes level:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: test-backup
  namespace: default
spec:
  source:
    volumeName: "test-volume"
    path: /mypath
  destination:
    path: /stashrepo
    repositorySecretName: test-secret
    volume:
      emptyDir: {}
      name: stash-volume
  schedule: "0 * * * * *"
  tags:
  - testTag
  retentionPolicy:
    keepLastSnapshots: 3
```

**Line 1-3**: With all other Kubernetes config, AppsCode Stash resource needs `apiVersion`, `kind` and `metadata` fields. `apiVersion` and `kind` needs to be exactly same as `stash.appscode.com/v1alpha1`, and, `specific version` currently as `v1alpha1`, to identify the resource
as AppsCode Stash. In metadata the `name` and `namespace` indicates the resource identifying name and its Kubernetes namespace.

**Line 6-20**: Stash spec has all the information needed to configure the backup process. 

* In `source` field user needs to specify the volume name and the path of which he/she wants to take backup.
* In `destination` field user needs to specify the path and the volume where he/she wants to store the backup snapshots.
* In `repositorySecretName` user adds name of the secret. In secret user will add a key `password`, which will be used as the password for the backup repository. Secret and Backup must be under the same namespace.
* User can add tag for the snapshots by using `tags` field. Multiple tags are allowed for a single snapshot.
* User needs to add the `schedule`. Its the time interval of taking snapshots. It will be in cron format. You can learn about cron format from [here](http://www.nncron.ru/help/EN/working/cron-format.htm).
* In `retentionPolicy` user adds the policy of keeping snapshots. Retention policy options are below.

```
# keepLastSnapshots: n --> never delete the n last (most recent) snapshots
# keepHourlySnapshots: n --> for the last n hours in which a snapshot was made, keep only the last snapshot for each hour.
# keepDailySnapshots: n --> for the last n days which have one or more snapshots, only keep the last one for that day.
# keepWeeklySnapshots: n --> for the last n weeks which have one or more snapshots, only keep the last one for that week.
# keepMonthlySnapshots: n --> for the last n months which have one or more snapshots, only keep the last one for that month.
# keepYearlySnapshots: n --> for the last n years which have one or more snapshots, only keep the last one for that year.
# keepTags --> keep all snapshots which have all tags specified by this option.
```                
One can restrict removing snapshots to those which have a particular hostname with the `retainHostname` , or tags with the `retainTags` option. 
When multiple `retainTags` are specified, only the snapshots which have all the tags are considered.

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

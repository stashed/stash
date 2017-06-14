# How to backup



## High Level Tasks
* Create `restik.backup.appscode.com` Third Party Resource
* Create Backup Deployment

## Deploying Backup

### Create Third Party Resource
`Backup process` depends on Third Party Resource Object `restik.backup.appscode.com`. This object can be created using following data.

```yaml
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
metadata:
  name: restik.backup.appscode.com
description: "Backup and restore support for Kubernetes persistent volumes by AppsCode"
versions:
  - name: v1alpha1
```


```sh
# Create Third Party Resource
$ kubectl apply -f https://raw.githubusercontent.com/appscode/restik/master/api/extensions/backup.yaml
```


### Deploy Controller
Restik controller communicates with kube-apiserver at inCluster mode if no master or kubeconfig is provided. It watches Restik resource to handle backup process.
```
$ kubectl apply -f https://raw.githubusercontent.com/appscode/restik/master/hack/deploy/deployments.yaml
```

#### Configuration Options
```
--master          // The address of the Kubernetes API server (overrides any value in kubeconfig)
--kubeconfig      // Path to kubeconfig file with authorization information (the master location is set by the master flag)
--image           // Restik image name with version to be run in restic-sidecar (appscode/restik:latest)
```

## Restik
This resource type is backed by a controller which take backup of kubernetes volumes from any running pod in Kubernetes. It can also take backup of host paths from Nodes in Kubernetes.

### Resource
A AppsCode Restik resource Looks like at the kubernetes level:

```yaml
apiVersion: backup.appscode.com/v1alpha1
kind: Restik
metadata:
  name: test-backup
  namespace: default
spec:
  source:
    volumeName: "test-volume"
    path: /mypath
  destination:
    path: /restikrepo
    repositorySecretName: test-secret
    volume:
      emptyDir: {}
      name: restik-volume
  schedule: "0 * * * * *"
  tags:
  - testTag
  retentionPolicy:
    keepLastSnapshots: 3
```

**Line 1-3**: With all other Kubernetes config, AppsCode Restik resource needs `apiVersion`, `kind` and `metadata` fields. `apiVersion` and `kind` needs to be exactly same as `backup.appscode.com/v1alpha1`, and, `specific version` currently as `v1alpha1`, to identify the resource
as AppsCode Restik. In metadata the `name` and `namespace` indicates the resource identifying name and its Kubernetes namespace.

**Line 6-20**: Restik spec has all the information needed to configure the backup process. 

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

For enabling the backup process for a particular kubernetes object like `RC`, `Replica Set`, `Deployment`, `DaemonSet` user adds a label `backup.appscode.com/config: <name_of_tpr>`. `<name_of_tpr>` is the name of Restik object. And then user creates the Restik object for starting backup process.
In case of StaefulSet user has to add the restic-sidecar container manually.

```yaml
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  labels:
    backup.appscode.com/config: test-backup
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
        - name: RESTIK_NAMESPACE
          value: default
        - name: TPR
          value: test-backup
        image: appscode/restik:latest
        imagePullPolicy: Always
        name: restic-sidecar
        volumeMounts:
        - mountPath: /source_path
          name: test-volume
        - mountPath: /repo_path
          name: restik-vol
      volumes:
      - emptyDir: {}
        name: test-volume
      - emptyDir: {}
        name: restik-vol
```

## Backup Nodes

If one interested in take backup of host paths, this can be done by deploying a `DaemonSet` with a do nothing busybox container. 
Restik TPR controller can use that as a vessel for running restic sidecar containers.

## Update Backup

One can update the source, retention policy, tags, cron schedule of the Restik object. After updating the Restik object backup process will follow the new backup strategy.
If user wants to update the image of restic-sidecar container he/she needs to update the `backup.appscode.com/image` in field annotation in the backup object. This will automatically update the restic-sidecar container.
In case of Statefulset user needs to update the sidecar container manually.

## Disable Backup

For disabling backup process one needs to delete the corresponding Restik object in case of `RC`, `Replica Set`, `Deployment`, `DaemonSet`.
In case of `Statefulset` user needs to delete the corresponding backup object as well as remove the side container from the Statefulset manually.
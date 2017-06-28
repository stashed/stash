# Tutorial




```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - command:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
      restartPolicy: Always
```

```sh
$ kubectl get pods -n default

NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-681367776-p8mff   1/1       Running   0          15s
```


```sh
$ kubectl create secret generic stash-demo --from-literal=RESTIC_PASSWORD=changeit
secret "stash-demo" created
```

You can check that the secret was created like this:

```yaml
$ kubectl get secret stash-demo -o yaml

apiVersion: v1
data:
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-06-28T08:17:00Z
  name: stash-demo
  namespace: default
  resourceVersion: "333"
  selfLink: /api/v1/namespaces/default/secrets/stash-demo
  uid: 28fe07e7-5bda-11e7-89db-080027bd2b24
type: Opaque
```


```yaml
$ kubectl get restic -n default stash-demo -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: stash-demo
  namespace: default
spec:
  backend:
    local:
      path: /repo
      volume:
        emptyDir: {}
        name: repo
    repositorySecretName: stash-demo
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
```






```sh
$ kubectl get pods -n default
NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-681367776-p8mff   2/2       Running   0          3m
```


```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    restic.appscode.com/config: stash-epggzp
    restic.appscode.com/tag: canary
  creationTimestamp: 2017-06-28T08:28:37Z
  generation: 2
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
  resourceVersion: "436"
  selfLink: /apis/extensions/v1beta1/namespaces/test-stash-dy4tec/deployments/stash-zjp2xq
  uid: c893e438-5bdb-11e7-8520-080027c24619
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: stash-demo
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - command:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      - args:
        - schedule
        - --v=3
        - --namespace=test-stash-dy4tec
        - --name=stash-epggzp
        - --app=stash-zjp2xq
        - --prefix-hostname=false
        image: appscode/stash:canary
        imagePullPolicy: Always
        name: stash
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /tmp
          name: stash-scratchdir
        - mountPath: /etc
          name: stash-podinfo
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
      volumes:
      - emptyDir: {}
        name: stash-scratchdir
      - downwardAPI:
          defaultMode: 420
          items:
          - fieldRef:
              apiVersion: v1
              fieldPath: metadata.labels
            path: labels
        name: stash-podinfo
status:
  conditions:
  - lastTransitionTime: 2017-06-28T08:28:37Z
    lastUpdateTime: 2017-06-28T08:28:37Z
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: 2017-06-28T08:28:38Z
    lastUpdateTime: 2017-06-28T08:28:38Z
    message: ReplicaSet "stash-zjp2xq-3019705014" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  replicas: 1
  unavailableReplicas: 1
  updatedReplicas: 1
```


```yaml
$ kubectl get restic stash-demo -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T08:37:48Z
  name: stash-demo
  namespace: default
  resourceVersion: "440"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/stash-demo
  uid: 10be2e8c-5bdd-11e7-9f08-08002778c951
spec:
  backend:
    local:
      path: /repo
      volume:
        emptyDir: {}
        name: repo
    repositorySecretName: stash-demo
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
status:
  backupCount: 3
  firstBackupTime: 2017-06-28T08:39:08Z
  lastBackupDuration: 1.575411972s
  lastBackupTime: 2017-06-28T08:41:08Z
```




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
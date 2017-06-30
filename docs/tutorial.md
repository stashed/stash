# Using Stash
This tutorial will show you how to use Stash to backup a Kubernetes deployment. At first, you need to have a Kubernetes cluster,
and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster,
you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install Stash in your cluster following the steps [here](/docs/install.md).

In this tutorial, we are going to backup the `/source/data` folder of a `busybox` pod into a local backend. First deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as source volume for demonstration purpose.

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
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      restartPolicy: Always
      volumes:
      - gitRepo:
          repository: https://github.com/appscode/stash-data.git
        name: source-data
```

```sh
$  kubectl create -f ./docs/examples/tutorial/busybox.yaml
deployment "stash-demo" created
```

Run the following command to confirm that `busybox` pods are running.

```sh
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-3651400299-0s1xb   1/1       Running   0          58s
```

Now, create a `Secret` that contains the key `RESTIC_PASSWORD`. This will be used as the password for your restic repository.

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

Now, create a `Restic` tpr with selectors matching the labels of the `busybox` Deployment. 

```sh
$ kubectl create -f ./docs/examples/tutorial/restic.yaml 
restic "stash-demo" created
```

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
      volume:
        emptyDir: {}
        name: safe-data
    repositorySecretName: stash-demo
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```

Here,
 - `spec.selector` is used to select workloads upon which this `Restic` configuration will be applied. `Restic` always selects workloads in the same Kubernetes namespace. In this tutorial, labels of `busybox` Deployment match this `Restic`'s selectors.
 - `spec.fileGroups` indicates an array of local paths that will be backed up using restic. For each path, users can also define the retention policy for old snapshots. Here, we are backing up the `/source/data` folder and only keeping the last 5 snapshots.
 - `spec.backend.local` indicates that restic will store the snapshots in a local path `/safe/data`. For the purpose of this tutorial, we are using an `emptyDir` to store the snapshots. But any Kubernetes volume that can be mounted locally can be used as a backend (example, NFS, Ceph, etc). Stash can also store snapshots in cloud storage solutions like, S3, GCS, Azure, etc.
  - `spec.backend.repositorySecretName` points to the Kubernetes secret created earlier in this tutorial. `Restic` always points to secrets in its own namespace. This secret is used to pass restic repository password and other cloud provider secrets to `restic` binary.
  - `spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates that file groups will be backed up every 1 minute.
  - `spec.volumeMounts` refers to volumes to be mounted in `stash` sidecar to get access to fileGroup path `/source/data`.

Stash operator watches for `Restic` objects using Kubernetes api. Stash operator will notice that the `busybox` Deployment matches the selector for `stash-demo` Restic object. So, it will add a sidecar container named `stash` to `busybox` Deployment and restart the running `busybox` pods. Since a local backend is used in `stash-demo` Restic, sidecar container will be mounted the corresponding persistent volume.

```sh
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-3001144127-3fsbn   2/2       Running   0          3m
```

```yaml
$ kubectl get deployment busybox -o yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    restic.appscode.com/config: stash-demo
    restic.appscode.com/tag: canary
  creationTimestamp: 2017-06-28T08:28:37Z
  generation: 2
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
  resourceVersion: "1703"
  selfLink: /apis/extensions/v1beta1/namespaces/default/deployments/stash-demo
  uid: e7fe819c-5d2c-11e7-bc7e-0800278d42f6
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
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      - args:
        - schedule
        - --restic-name=stash-demo
        - --workload=Deployment/stash-demo
        - --v=3
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        image: appscode/stash:0.2.0
        imagePullPolicy: IfNotPresent
        name: stash
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /tmp
          name: stash-scratchdir
        - mountPath: /etc/stash
          name: stash-podinfo
        - mountPath: /source/data
          name: source-data
          readOnly: true
        - mountPath: /safe/data
          name: safe-data
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
      volumes:
      - gitRepo:
          repository: https://github.com/appscode/stash-data.git
        name: source-data
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
      - emptyDir: {}
        name: safe-data
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
    message: ReplicaSet "stash-demo-3019705014" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  replicas: 1
  unavailableReplicas: 1
  updatedReplicas: 1
```

Now, wait a few minutes so that restic can take a backup of the `/source/data` folder. To confirm, check the `status.backupCount` of `stash-demo` Restic tpr.

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
      volume:
        emptyDir: {}
        name: safe-data
    repositorySecretName: stash-demo
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
status:
  backupCount: 1
  firstBackupTime: 2017-06-28T08:39:08Z
  lastBackupDuration: 1.575411972s
  lastBackupTime: 2017-06-28T08:39:08Z
```

You can also exec into the `busybox` Deployment to check list of snapshots.

```sh
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-3001144127-3fsbn   2/2       Running   0          49s

$ kubectl exec -it stash-demo-3001144127-3fsbn -c stash sh
/ # export RESTIC_REPOSITORY=/repo
/ # export RESTIC_PASSWORD=changeit
/ # restic snapshots
ID        Date                 Host                         Tags        Directory
----------------------------------------------------------------------
c275bb54  2017-06-28 08:39:08  stash-demo-3001144127-3fsbn              /source/data
```

## Disable Backup
To stop taking backup of `/source/data` folder, delete the `stash-demo` Restic tpr. As a result, Stash operator will remove the sidecar container from `busybox` Deployment.
```sh
$ kubectl delete restic stash-demo
restic "stash-demo" deleted

$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS        RESTARTS   AGE
stash-demo-3001144127-3fsbn   2/2       Terminating   0          3m
stash-demo-3651400299-8c14s   1/1       Running       0          5s
```

## Cleaning up
To cleanup the Kubernetes resources created by this tutorial, run:
```sh
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/uninstall.md).


## Next Steps
- Learn about the details of Restic tpr [here](/docs/concept.md).
- See the list of supported backends and how to configure them [here](/docs/backends.md).
- See working examples for supported workload types [here](/docs/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/monitoring.md).
- Wondering what features are coming next? Please visit [here](/ROADMAP.md). 
- Want to hack on Stash? Check our [contribution guidelines](/CONTRIBUTING.md).

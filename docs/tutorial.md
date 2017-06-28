# Using Stash
This tutorial will show you how to use Stash to backup a Kubernetes deployment. To start, install Stash in your cluster following the steps [here](/docs/install.md). This tutorial can be run using [minikube](https://github.com/kubernetes/minikube).

In this tutorial, we are going to backup the `/lib` folder of a `busybox` pod into a local backend. First deploy the following `busybox` Deployment in your cluster.

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

```sh
$ kubectl create -f ./docs/examples/tutorial/restic.yaml 
restic "stash-demo" created
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


## Cleaning up
To cleanup the Kubernetes resources created by this tutorial, run:
```sh
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/uninstall.md).

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
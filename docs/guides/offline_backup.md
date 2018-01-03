---
title: Offline Backup | Stash
description: Offline Backup using Stash
menu:
  product_stash_0.6.0:
    identifier: offline-stash
    name: Offline Backup
    parent: guides
    weight: 15
product_name: stash
menu_name: product_stash_0.6.0
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Offline Backup

This tutorial will show you how to backup a Kubernetes deployment using Stash in offline mode. By default, stash takes backup in [online](/docs/guides/backup.md) mode where sidecar container is added to take periodic backups and check backups. But sometimes you need to ensure that source data is not being modified while taking backup, that means running backup while keeping workload pod stopped. In such case you can run backup in offline mode. To do this you need to specify `spec.type=offline` in `Restic` CRD.

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install Stash in your cluster following the steps [here](/docs/setup/install.md).

In this tutorial, we are going to backup the `/source/data` folder of a `busybox` pod into a local backend. First deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as source volume for demonstration purpose.

```yaml
apiVersion: apps/v1beta1
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

```console
$  kubectl apply -f ./docs/examples/tutorial/busybox.yaml
deployment "stash-demo" created
```

Run the following command to confirm that `busybox` pods are running.

```console
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-788ffcf9c6-p5kxc   1/1       Running   0          12s
```

Now, create a `Secret` that contains the key `RESTIC_PASSWORD`. This will be used as the password for your restic repository.

```console
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
  creationTimestamp: 2017-12-04T05:24:22Z
  name: stash-demo
  namespace: default
  resourceVersion: "22328"
  selfLink: /api/v1/namespaces/default/secrets/stash-demo
  uid: 62aa8ef8-d8b3-11e7-be92-0800277f19c0
type: Opaque
```

Now, create a `Restic` CRD with selectors matching the labels of the `busybox` Deployment and `spec.type=offline`.

```console
$ kubectl apply -f ./docs/examples/tutorial/restic_offline.yaml
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
  type: offline
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  schedule: '@every 5m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

When a `Restic` is created with `spec.type=offline`, stash operator add a [init-container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) instead of sidecar container to target workload pods. The init-container takes backup once. If backup is successfully completed, then it creates a job to perform `restic check` and exits. The app container starts only after the init-container exits without any error. This ensures that the app container is not running while taking backup.
Stash operator also creates a cron-job that deletes the workload pods according to the `spec.schedule`. Thus the workload pods get restarted periodically and allows the init-container to take backup.

```console
$ kubectl get pods -l app=stash-demo -w
NAME                          READY     STATUS        RESTARTS   AGE
stash-demo-788ffcf9c6-p5kxc   1/1       Terminating   0          1m
stash-demo-7b4f6877dc-nhrz9   0/1       Init:0/1      0          4s
stash-demo-7b4f6877dc-nhrz9   1/1       Running       0          32s
```

```yaml
$ kubectl get deployment stash-demo -o yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    restic.appscode.com/last-applied-configuration: |
      {"kind":"Restic","apiVersion":"stash.appscode.com/v1alpha1","metadata":{"name":"stash-demo","namespace":"default","selfLink":"/apis/stash.appscode.com/v1alpha1/namespaces/default/restics/stash-demo","uid":"c55d5918-d8da-11e7-be92-0800277f19c0","resourceVersion":"57719","creationTimestamp":"2017-12-04T10:06:18Z"},"spec":{"selector":{"matchLabels":{"app":"stash-demo"}},"fileGroups":[{"path":"/source/data","retentionPolicyName":"keep-last-5"}],"backend":{"storageSecretName":"stash-demo","local":{"volumeSource":{"hostPath":{"path":"/data/stash-test/restic-repo"}},"path":"/safe/data"}},"schedule":"@every 5m","volumeMounts":[{"name":"source-data","mountPath":"/source/data"}],"resources":{},"retentionPolicies":[{"name":"keep-last-5","keepLast":5,"prune":true}],"type":"offline"},"status":{}}
    restic.appscode.com/tag: canary
  creationTimestamp: 2017-12-04T10:04:11Z
  generation: 2
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
  resourceVersion: "57824"
  selfLink: /apis/extensions/v1beta1/namespaces/default/deployments/stash-demo
  uid: 798d4e60-d8da-11e7-be92-0800277f19c0
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: stash-demo
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
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
      dnsPolicy: ClusterFirst
      initContainers:
      - args:
        - backup
        - --restic-name=stash-demo
        - --workload-kind=Deployment
        - --workload-name=stash-demo
        - --image-tag=canary
        - --enable-rbac=true
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
        image: appscode/stash:0.6.0
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
          name: stash-local
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
      - hostPath:
          path: /data/stash-test/restic-repo
          type: ""
        name: stash-local
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: 2017-12-04T10:06:26Z
    lastUpdateTime: 2017-12-04T10:06:26Z
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: 2017-12-04T10:04:11Z
    lastUpdateTime: 2017-12-04T10:06:26Z
    message: ReplicaSet "stash-demo-7b4f6877dc" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```

Now, wait a few minutes so that restic can take a backup of the `/source/data` folder. To confirm, check the `status.backupCount` of `stash-demo` Restic CRD.

```yaml
$ kubectl get restic stash-demo -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  clusterName: ""
  creationTimestamp: 2017-12-04T10:06:18Z
  deletionGracePeriodSeconds: null
  deletionTimestamp: null
  generation: 0
  initializers: null
  name: stash-demo
  namespace: default
  resourceVersion: "57790"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/stash-demo
  uid: c55d5918-d8da-11e7-be92-0800277f19c0
spec:
  backend:
    local:
      mountPath: /safe/data
      volumeSource:
        hostPath:
          path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: keep-last-5
  retentionPolicies:
  - keepLast: 5
    name: keep-last-5
    prune: true
  schedule: '@every 5m'
  selector:
    matchLabels:
      app: stash-demo
  type: offline
  volumeMounts:
  - mountPath: /source/data
    name: source-data
status:
  backupCount: 1
  firstBackupTime: 2017-12-04T10:06:23Z
  lastBackupDuration: 1.5347814s
  lastBackupTime: 2017-12-04T10:06:23Z
```

Stash operator also creates a cron job to periodically delete workload pods according to `spec.schedule`. Please note that Kubernetes cron jobs [do not support timezone](https://github.com/kubernetes/kubernetes/issues/47202).

```console
kubectl get cronjob
NAME                            SCHEDULE    SUSPEND   ACTIVE    LAST SCHEDULE   AGE
stash-kubectl-cron-stash-demo   @every 5m   False     0         <none>
```

Note that offline backup is not supported for workload kind `Deployment`, `Replicaset` and `ReplicationController` with `replicas > 1`.

## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:
```console
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Learn about how to configure Stash operator as workload initializer [here](/docs/guides/initializer.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).
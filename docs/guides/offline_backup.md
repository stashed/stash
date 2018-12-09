---
title: Offline Backup | Stash
description: Offline Backup using Stash
menu:
  product_stash_0.8.1:
    identifier: offline-stash
    name: Offline Backup
    parent: guides
    weight: 15
product_name: stash
menu_name: product_stash_0.8.1
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Offline Backup

This tutorial will show you how to backup a Kubernetes deployment using Stash in offline mode. By default, stash takes backup in [online](/docs/guides/backup.md) mode where sidecar container is added to take periodic backups and check backups. But sometimes you need to ensure that source data is not being modified while taking the backup, that means running backup while keeping workload pod stopped. In such case, you can run the backup in offline mode. To do this you need to specify `spec.type: offline` in `Restic` crd.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the `kubectl` command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

- Install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

- You should be familiar with the following Stash concepts:
  - [Restic](/docs/concepts/crds/restic.md)
  - [Repository](/docs/concepts/crds/repository.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

- You will need an NFS server to store backed up data. If you already do not have an NFS server running, deploy one following the tutorial from [here](https://github.com/appscode/third-party-tools/blob/master/storage/nfs/README.md). For this tutorial, we have deployed NFS server in `storage` namespace and it is accessible through `nfs-service.storage.svc.cluster.local` dns.

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/backup](/docs/examples/backup) directory of [appscode/stash](https://github.com/appscode/stash) repository.

## Overview

The following diagram shows how Stash takes offline backup of a Kubernetes volume. Open the image in a new tab to see the enlarged image.

<p align="center">
  <img alt="Stash Offline Backup Flow" src="/docs/images/stash-offline-backup.svg">
</p>

The offline backup process consists of the following steps:

1. At first, a user creates a `Secret`. This secret holds the credentials to access the backend where backed up data will be stored. It also holds a password (`RESTIC_PASSWORD`) that will be used to encrypt the backed up data.
2. Then, the user creates a `Restic` crd which specifies the targeted workload for backup. It also specifies the backend information where the backed up data will be stored.
3. Stash operator watches for `Restic` crd. Once it sees a `Restic` crd, it identifies the targeted workload that matches the selector of this `Restic`.
4. Then, Stash operator injects an [init-container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) named `stash` and mounts the target volume in it.
5. Stash operator creates a [CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) with name `stash-scaledown-cron-{restic-name}`.
6. The `CronJob` restarts workload on the scheduled interval.
7. Finally, `stash` init-container takes backup of the volume to the specified backend when pod restarts. It also creates a `Repository` crd during the first backup which represents the backend in Kubernetes native way.

The `CronJob` restarts workloads according to the following rules:

1. If the workload is a [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) or a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/), it will delete all pods of the workload. The workload will automatically re-creates the pods and each pod will take backup with their `init-container`.
2. If the workload is a [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [ReplicaSet](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/) or [ReplicationController](https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller/), it will scale down the workload to 0 replica. When all pods are terminated, it will scale up the workload to 1 replica. This single replica will take backup with `init-container`. When backup is complete, the `init-container` will scale up the workload to original replica. The rest of the replicas will not take backup even through they have `init-container`.

## Backup

In order to take backup, we need some sample data. Stash has some sample data in [appscode/stash-data](https://github.com/appscode/stash-data) repository. As [gitRepo](https://kubernetes.io/docs/concepts/storage/volumes/#gitrepo) volume has been deprecated, we are not going to use this repository as volume directly. Instead, we are going to create a [configMap](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) from the stash-data repository and use that ConfigMap as data source.

Let's create a ConfigMap from these sample data,

```console
$ kubectl create configmap -n demo stash-sample-data \
	--from-literal=LICENSE="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash-data/master/LICENSE)" \
	--from-literal=README.md="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash-data/master/README.md)"
configmap/stash-sample-data created
```

Here, we are going to backup the `/source/data` folder of a `busybox` pod into an [NFS](https://kubernetes.io/docs/concepts/storage/volumes/#nfs) volume. NFS volume is a type of [local](/docs/guides/backends/local.md) backend for Stash.

**Deploy Workload:**

Now, deploy the following Deployment. Here, we have mounted the ConfigMap `stash-sample-data` as data source volume.

Below, the YAML for the Deployment we are going to create.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: stash-demo
  name: stash-demo
  namespace: demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: stash-demo
  template:
    metadata:
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - args:
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
      - name: source-data
        configMap:
          name: stash-sample-data
```

Let's create the deployment we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/deployment.yaml
deployment.apps/stash-demo created
```

Now, wait for deployment's pod to go into `Running` state.

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-7ccd56bf5d-p9p2p   1/1     Running   0          2m29s
```

You can check that the `/source/data/` directory of this pod is populated with data from the `stash-sample-data` ConfigMap using this command,

```console
$ kubectl exec -n demo stash-demo-7ccd56bf5d-p9p2p -- ls -R /source/data
/source/data:
LICENSE
README.md
```

Now, we are ready to backup `/source/data` directory into an NFS backend.

**Create Secret:**

At first, we need to create a storage secret. To configure this backend, the following secret keys are needed:

|        Key        |                        Description                         |
| ----------------- | ---------------------------------------------------------- |
| `RESTIC_PASSWORD` | `Required`. Password used to encrypt snapshots by `restic` |

Create the secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ kubectl create secret generic -n demo local-secret \
    --from-file=./RESTIC_PASSWORD
secret/local-secret created
```

Verify that the secret has been created successfully.

```console
$ kubectl get secret -n demo local-secret -o yaml
```

```yaml
apiVersion: v1
data:
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2018-12-07T11:44:09Z
  name: local-secret
  namespace: demo
  resourceVersion: "36409"
  selfLink: /api/v1/namespaces/demo/secrets/local-secret
  uid: 68ab5960-fa15-11e8-8905-0800277ca39d
type: Opaque
```

**Create Restic:**

Now, we are going to create `Restic` crd to take backup `/source/data` directory of `stash-demo` deployment in offline mode.

Below, the YAML for Restic crd we are going to create for offline backup,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: offline-restic
  namespace: demo
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
      nfs:
        server: "nfs-service.storage.svc.cluster.local"
        path: "/"
    storageSecretName: local-secret
  schedule: '@every 5m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Here, we have set `spec.type: offline`. This tell Stash to take backup in offline mode.

Let's create the `Restic` we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/restic_offline.yaml
restic.stash.appscode.com/offline-restic created
```

If everything goes well, Stash will inject an [init-container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) into the `stash-demo` deployment to take backup while pod starts.

Let's check that `init-container` has been injected successfully,

```console
$ kubectl get deployment -n demo stash-demo -o yaml
```

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"stash-demo"},"name":"stash-demo","namespace":"demo"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"stash-demo"}},"template":{"metadata":{"labels":{"app":"stash-demo"},"name":"busybox"},"spec":{"containers":[{"args":["sleep","3600"],"image":"busybox","imagePullPolicy":"IfNotPresent","name":"busybox","volumeMounts":[{"mountPath":"/source/data","name":"source-data"}]}],"restartPolicy":"Always","volumes":[{"configMap":{"name":"stash-sample-data"},"name":"source-data"}]}}}}
    restic.appscode.com/last-applied-configuration: |
      {"kind":"Restic","apiVersion":"stash.appscode.com/v1alpha1","metadata":{"name":"offline-restic","namespace":"demo","selfLink":"/apis/stash.appscode.com/v1alpha1/namespaces/demo/restics/offline-restic","uid":"f5b3abe7-fa15-11e8-8905-0800277ca39d","resourceVersion":"36693","generation":1,"creationTimestamp":"2018-12-07T11:48:05Z","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"stash.appscode.com/v1alpha1\",\"kind\":\"Restic\",\"metadata\":{\"annotations\":{},\"name\":\"offline-restic\",\"namespace\":\"demo\"},\"spec\":{\"backend\":{\"local\":{\"mountPath\":\"/safe/data\",\"nfs\":{\"path\":\"/\",\"server\":\"nfs-service.storage.svc.cluster.local\"}},\"storageSecretName\":\"local-secret\"},\"fileGroups\":[{\"path\":\"/source/data\",\"retentionPolicyName\":\"keep-last-5\"}],\"retentionPolicies\":[{\"keepLast\":5,\"name\":\"keep-last-5\",\"prune\":true}],\"schedule\":\"@every 5m\",\"selector\":{\"matchLabels\":{\"app\":\"stash-demo\"}},\"type\":\"offline\",\"volumeMounts\":[{\"mountPath\":\"/source/data\",\"name\":\"source-data\"}]}}\n"}},"spec":{"selector":{"matchLabels":{"app":"stash-demo"}},"fileGroups":[{"path":"/source/data","retentionPolicyName":"keep-last-5"}],"backend":{"storageSecretName":"local-secret","local":{"nfs":{"server":"nfs-service.storage.svc.cluster.local","path":"/"},"mountPath":"/safe/data"}},"schedule":"@every 5m","volumeMounts":[{"name":"source-data","mountPath":"/source/data"}],"resources":{},"retentionPolicies":[{"name":"keep-last-5","keepLast":5,"prune":true}],"type":"offline"}}
    restic.appscode.com/tag: e3
  creationTimestamp: 2018-12-07T11:40:30Z
  generation: 2
  labels:
    app: stash-demo
  name: stash-demo
  namespace: demo
  resourceVersion: "36735"
  selfLink: /apis/extensions/v1beta1/namespaces/demo/deployments/stash-demo
  uid: e6996fbd-fa14-11e8-8905-0800277ca39d
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
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
      annotations:
        restic.appscode.com/resource-hash: "16527601205197612609"
      creationTimestamp: null
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - args:
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
        - --restic-name=offline-restic
        - --workload-kind=Deployment
        - --workload-name=stash-demo
        - --docker-registry=appscodeci
        - --image-tag=e3
        - --pushgateway-url=http://stash-operator.kube-system.svc:56789
        - --enable-status-subresource=true
        - --use-kubeapiserver-fqdn-for-aks=true
        - --enable-analytics=true
        - --logtostderr=true
        - --alsologtostderr=false
        - --v=3
        - --stderrthreshold=0
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
        - name: APPSCODE_ANALYTICS_CLIENT_ID
          value: 90b12fedfef2068a5f608219d5e7904a
        image: appscodeci/stash:e3
        imagePullPolicy: IfNotPresent
        name: stash
        resources: {}
        securityContext:
          procMount: Default
          runAsUser: 0
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
      - configMap:
          defaultMode: 420
          name: stash-sample-data
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
      - name: stash-local
        nfs:
          path: /
          server: nfs-service.storage.svc.cluster.local
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: 2018-12-07T11:42:45Z
    lastUpdateTime: 2018-12-07T11:42:45Z
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: 2018-12-07T11:40:31Z
    lastUpdateTime: 2018-12-07T11:48:08Z
    message: ReplicaSet "stash-demo-684cd86f7b" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```

Notice that `stash-demo` deployment has an `init-container` named `stash` which is running `backup` command.

Stash operator also has created a [CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) with name format `stash-scaledown-cron-{restic-name}`. Verify that the `CronJob` has been created successfully,

```console
$ kubectl get cronjob -n demo
NAME                                  SCHEDULE    SUSPEND   ACTIVE   LAST SCHEDULE   AGE
stash-scaledown-cron-offline-restic   @every 5m   False     0        <none>          2m34s
```

**Verify Backup:**

Stash will create a `Repository` crd with name `deployment.stash-demo` for the respective repository during the first backup run. To verify, run the following command,

```console
$  kubectl get repository deployment.stash-demo -n demo
NAME                    BACKUP-COUNT  LAST-SUCCESSFUL-BACKUP   AGE
deployment.stash-demo   1             2m                       2m
```

Here, `BACKUP-COUNT` field indicates number of backup snapshot has taken in this repository.

## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete -n demo deployment stash-demo
$ kubectl delete -n demo secret local-secret
$ kubectl delete -n demo restic offline-restic
$ kubectl delete -n demo repository deployment.stash-demo

$ kubectl delete namespace demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- See the list of supported backends and how to configure them [here](/docs/guides/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

---
title: Backup Volumes | Stash
description: Backup Volumes using Stash
menu:
  product_stash_0.8.3:
    identifier: backup-stash
    name: Backup Volumes
    parent: v1alpha1-guides
    weight: 10
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Backup Volumes using Stash

This tutorial will show you how to use Stash to back up a Kubernetes volume. Here, we are going to backup the `/source/data` folder of a busybox pod into an [NFS](https://kubernetes.io/docs/concepts/storage/volumes/#nfs) volume. NFS volume is configured as a [Local](/docs/guides/v1alpha1/backends/local.md) backend of Stash.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the `kubectl` command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

- Install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

- You should be familiar with the following Stash concepts:

  - [Restic](/docs/concepts/crds/v1alpha1/restic.md)
  - [Repository](/docs/concepts/crds/repository.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

- You will need an NFS server to store backed up data. If you already do not have an NFS server running, deploy one following the tutorial from [here](https://github.com/appscode/third-party-tools/blob/master/storage/nfs/README.md). For this tutorial, we have deployed NFS server in `storage` namespace and it is accessible through `nfs-service.storage.svc.cluster.local` dns.

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/backup](/docs/examples/backup) directory of [appscode/stash](https://github.com/stashed/stash) repository.

## Overview

The following diagram shows how Stash takes backup of a Kubernetes volume. Open the image in a new tab to see the enlarged image.

<p align="center">
  <img alt="Stash Backup Flow" src="/docs/images/v1alpha1/stash-backup.svg">
</p>

The backup process consists of the following steps:

1. At first, a user creates a `Secret`. This secret holds the credentials to access the backend where backed up data will be stored. It also holds a password (`RESTIC_PASSWORD`) that will be used to encrypt the backed up data.
2. Then, the user creates a `Restic` crd which specifies the targeted workload for backup. It also specifies the backend information where the backed up data will be stored.
3. Stash operator watches for `Restic` crd. Once, it found a `Restic` crd, it identifies the targeted workloads that match `Restic`'s selector.
4. Then, Stash operator injects a sidecar container named `stash` and mounts the target volume into it.
5. Finally, `stash` sidecar container takes periodic backup of the volume to specified backend. It also creates a `Repository` crd in first backup which represents the original repository in the backend in a Kubernetes native way.

## Backup

In order to take back up, we need some sample data. Stash has some sample data in [appscode/stash-data](https://github.com/stashed/stash-data) repository. As [gitRepo](https://kubernetes.io/docs/concepts/storage/volumes/#gitrepo) volume has been deprecated, we are not going to use this repository volume directly. Instead, we are going to create a [configMap](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) from these data and use that ConfigMap as the data source.

Let's create a ConfigMap from these sample data,

```console
$ kubectl create configmap -n demo stash-sample-data \
	--from-literal=LICENSE="$(curl -fsSL https://raw.githubusercontent.com/stashed/stash-data/master/LICENSE)" \
	--from-literal=README.md="$(curl -fsSL https://raw.githubusercontent.com/stashed/stash-data/master/README.md)"
configmap/stash-sample-data created
```

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
stash-demo-7ccd56bf5d-4x27d   1/1     Running   0          21s
```

You can check that the `/source/data/` directory of this pod is populated with data from the `stash-sample-data` ConfigMap using this command,

```console
$ kubectl exec -n demo stash-demo-7ccd56bf5d-4x27d -- ls -R /source/data
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

Verify that the secret has been created successfully,

```console
$ kubectl get secret -n demo local-secret -o yaml
```

```yaml
apiVersion: v1
data:
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2018-12-07T06:04:56Z
  name: local-secret
  namespace: demo
  resourceVersion: "6049"
  selfLink: /api/v1/namespaces/demo/secrets/local-secret
  uid: 05a8d2a3-f9e6-11e8-8905-0800277ca39d
type: Opaque
```

**Create Restic:**

Now, we are going to create a `Restic` crd to back up `/source/data` directory of `stash-demo` deployment. This will create a repository in the directory of NFS server specified by `local.nfs.path` field and start taking periodic backup of `/source/data` directory.

Below, the YAML for Restic crd we are going to create,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: local-restic
  namespace: demo
spec:
  selector:
    matchLabels:
      app: stash-demo
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
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Here,

 - `spec.selector` is used to select workloads upon which this `Restic` configuration will be applied. `Restic` always selects workloads in the same Kubernetes namespace. In this tutorial, labels of `stash-demo` Deployment match this `Restic`'s selectors. If multiple `Restic` objects are matched to a given workload, Stash operator will error out and avoid adding sidecar container.
 - `spec.retentionPolicies` defines an array of retention policies, which can be used in `fileGroups` using `retentionPolicyName`.
 - `spec.fileGroups` indicates an array of local paths that will be backed up using restic. For each path, users can also specify the retention policy for old snapshots using `retentionPolicyName`, which must be defined in `spec.retentionPolicies`. Here, we are backing up the `/source/data` folder and only keeping the last 5 snapshots.
 - `spec.backend.local` indicates that restic will store the snapshots in a local path `/safe/data`. For the purpose of this tutorial, we are using an `NFS` server to store the snapshots. But any Kubernetes volume that can be mounted locally can be used as a backend (i.e. `hostPath`, `Ceph` etc). Stash can also store snapshots in cloud storage solutions like S3, GCS, Azure, etc. To use a remote backend, you need to configure the storage secret to include your cloud provider credentials and set one of `spec.backend.(s3|gcs|azure|swift|b2)`. Please visit [here](/docs/guides/v1alpha1/backends/overview.md) for more detailed examples.

  - `spec.backend.storageSecretName` points to the Kubernetes secret created earlier in this tutorial. `Restic` always points to secrets in its own namespace. This secret is used to pass restic repository password and other cloud provider secrets to `restic` binary.
  - `spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates that file groups will be backed up every 1 minute.
  - `spec.volumeMounts` refers to volumes to be mounted in `stash` sidecar to get access to fileGroup path `/source/data`.

Let's create the `Restic` we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/restic.yaml
restic.stash.appscode.com/local-restic created
```

If everything goes well, Stash will inject a sidecar container into the `stash-demo` deployment to take periodic backup. Let's check that sidecar has been injected successfully,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-7ffdb5d7fd-5x8l6   2/2     Running   0          37s
```

Look at the pod. It now has 2 containers. If you view the resource definition of this pod, you will see that there is a container named `stash` which running `backup` command.

```console
$ kubectl get pod -n demo stash-demo-7ffdb5d7fd-5x8l6 -o yaml
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    restic.appscode.com/resource-hash: "7515193209300432018"
  creationTimestamp: 2018-12-07T06:23:00Z
  generateName: stash-demo-7ffdb5d7fd-
  labels:
    app: stash-demo
    pod-template-hash: 7ffdb5d7fd
  name: stash-demo-7ffdb5d7fd-5x8l6
  namespace: demo
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: stash-demo-7ffdb5d7fd
    uid: 8bbc5b0e-f9e8-11e8-8905-0800277ca39d
  resourceVersion: "7496"
  selfLink: /api/v1/namespaces/demo/pods/stash-demo-7ffdb5d7fd-5x8l6
  uid: 8bc19dc8-f9e8-11e8-8905-0800277ca39d
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
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-6dqgm
      readOnly: true
  - args:
    - backup
    - --restic-name=local-restic
    - --workload-kind=Deployment
    - --workload-name=stash-demo
    - --docker-registry=appscodeci
    - --image-tag=e3
    - --run-via-cron=true
    - --pushgateway-url=http://stash-operator.kube-system.svc:56789
    - --enable-status-subresource=true
    - --use-kubeapiserver-fqdn-for-aks=true
    - --enable-analytics=true
    - --enable-rbac=true
    - --logtostderr=true
    - --alsologtostderr=false
    - --v=3
    - --stderrthreshold=0
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
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-6dqgm
      readOnly: true
  dnsPolicy: ClusterFirst
  nodeName: minikube
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
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
  - name: default-token-6dqgm
    secret:
      defaultMode: 420
      secretName: default-token-6dqgm
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: 2018-12-07T06:23:00Z
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: 2018-12-07T06:23:02Z
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: 2018-12-07T06:23:02Z
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: 2018-12-07T06:23:00Z
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://ba9282c73548f2c7e9e34313198c17814cfceaa60f2712547dfd8bcb40f8d4dc
    image: busybox:latest
    imageID: docker-pullable://busybox@sha256:2a03a6059f21e150ae84b0973863609494aad70f0a80eaeb64bddd8d92465812
    lastState: {}
    name: busybox
    ready: true
    restartCount: 0
    state:
      running:
        startedAt: 2018-12-07T06:23:01Z
  - containerID: docker://81afe30d602fa1a39d33bef894d7f4c67386d4c2a5c09afcfb8d1f10c6f63bf5
    image: appscodeci/stash:e3
    imageID: docker-pullable://appscodeci/stash@sha256:1e965663d00280a14cebb926f29d95547b746e6060c9aaaef649664f4600ffbe
    lastState: {}
    name: stash
    ready: true
    restartCount: 0
    state:
      running:
        startedAt: 2018-12-07T06:23:01Z
  hostIP: 10.0.2.15
  phase: Running
  podIP: 172.17.0.7
  qosClass: BestEffort
  startTime: 2018-12-07T06:23:00Z
```

**Verify Backup:**

Stash will create a `Repository` crd with name `deployment.stash-demo` for the respective repository in local backend at first backup schedule. To verify, run the following command,

```console
$  kubectl get repository deployment.stash-demo -n demo
NAME                    BACKUPCOUNT   LASTSUCCESSFULBACKUP   AGE
deployment.stash-demo   4             23s                    4m
```

Here, `BACKUPCOUNT` field indicates the number of backup snapshots has taken in this repository.

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup snapshots have been created successfully by running the following command:

```console
$ kubectl get snapshots -n demo -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-9a6e6b78   3m18s
deployment.stash-demo-2da5b6bc   2m18s
deployment.stash-demo-0f89f60e   78s
deployment.stash-demo-f9c704e4   18s
```

Here, we can see 4 last successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

## Disable Backup

To stop Stash from taking backup, you can do following things:

- Set `spec.paused: true` in Restic `yaml` and then apply the update.
This means:

  - Paused Restic CRDs will not applied to newly created workloads.
  - Stash sidecar containers will not be removed from existing workloads but the sidecar will stop taking backup.

```command
$ kubectl patch restic -n demo local-restic --type="merge" --patch='{"spec": {"paused": true}}'
restic.stash.appscode.com/local-restic patched
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: local-restic
  namespace: demo
spec:
  selector:
    matchLabels:
      app: stash-demo
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
  schedule: '@every 1m'
  paused: true
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

- Delete the Restic CRD. Stash operator will remove the sidecar container from all matching workloads.

  ```commands
  $ kubectl delete restic -n demo local-restic
  restic.stash.appscode.com/local-restic deleted
  ```

- Change the labels of a workload. Stash operator will remove sidecar container from that workload. This way you can selectively stop backup of a Deployment, ReplicaSet etc.

### Resume Backup

You can resume Restic to backup by setting `spec.paused: false` in Restic `yaml` and applying the update or you can patch Restic using,

```command
$ kubectl patch restic -n demo local-restic --type="merge" --patch='{"spec": {"paused": false}}'
restic.stash.appscode.com/local-restic patched
```

## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete -n demo deployment stash-demo
$ kubectl delete -n demo secret local-secret
$ kubectl delete -n demo restic local-restic
$ kubectl delete -n demo repository deployment.stash-demo

$ kubectl delete ns demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn about the details of Restic CRD [here](/docs/concepts/crds/v1alpha1/restic.md).
- To restore a backup see [here](/docs/guides/v1alpha1/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/v1alpha1/recovery.md).
- To run backup in offline mode see [here](/docs/guides/v1alpha1/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/v1alpha1/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/v1alpha1/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/v1alpha1/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/v1alpha1/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

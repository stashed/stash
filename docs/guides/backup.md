---
title: Backup Volumes | Stash
description: Backup Volumes using Stash
menu:
  product_stash_0.7.0-rc.3:
    identifier: backup-stash
    name: Backup Volumes
    parent: guides
    weight: 10
product_name: stash
menu_name: product_stash_0.7.0-rc.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Backup

This tutorial will show you how to use Stash to backup a Kubernetes deployment. At first, you need to have a Kubernetes cluster,
and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster,
you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install Stash in your cluster following the steps [here](/docs/setup/install.md).

In this tutorial, we are going to backup the `/source/data` folder of a `busybox` pod into a local backend. First deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as source volume for demonstration purpose.

```console
$  kubectl apply -f ./docs/examples/tutorial/busybox.yaml
deployment "stash-demo" created
```

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

Run the following command to confirm that `busybox` pods are running.

```console
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-788ffcf9c6-6t6lj   1/1       Running   0          12s
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

Now, create a `Restic` CRD with selectors matching the labels of the `busybox` Deployment.

```console
$ kubectl apply -f ./docs/examples/tutorial/restic.yaml
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
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
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

 - `spec.selector` is used to select workloads upon which this `Restic` configuration will be applied. `Restic` always selects workloads in the same Kubernetes namespace. In this tutorial, labels of `busybox` Deployment match this `Restic`'s selectors. If multiple `Restic` objects are matched to a given workload, Stash operator will error out and avoid adding sidecar container.
 - `spec.retentionPolicies` defines an array of retention policies, which can be used in `fileGroups` using `retentionPolicyName`.
 - `spec.fileGroups` indicates an array of local paths that will be backed up using restic. For each path, users can also specify the retention policy for old snapshots using `retentionPolicyName`, which must be defined in `spec.retentionPolicies`. Here, we are backing up the `/source/data` folder and only keeping the last 5 snapshots.
 - `spec.backend.local` indicates that restic will store the snapshots in a local path `/safe/data`. For the purpose of this tutorial, we are using an `hostPath` to store the snapshots. But any Kubernetes volume that can be mounted locally can be used as a backend (example, NFS, Ceph, etc). Stash can also store snapshots in cloud storage solutions like S3, GCS, Azure, etc. To use a remote backend, you need to configure the storage secret to include your cloud provider credentials and set one of `spec.backend.(s3|gcs|azure|swift|b2)`. Please visit [here](/docs/guides/backends.md) for more detailed examples.

  - `spec.backend.storageSecretName` points to the Kubernetes secret created earlier in this tutorial. `Restic` always points to secrets in its own namespace. This secret is used to pass restic repository password and other cloud provider secrets to `restic` binary.
  - `spec.schedule` is a [cron expression](https://github.com/robfig/cron/blob/v2/doc.go#L26) that indicates that file groups will be backed up every 1 minute.
  - `spec.volumeMounts` refers to volumes to be mounted in `stash` sidecar to get access to fileGroup path `/source/data`.

Stash operator watches for `Restic` objects using Kubernetes api. Stash operator will notice that the `busybox` Deployment matches the selector for `stash-demo` Restic object. So, it will add a sidecar container named `stash` to `busybox` Deployment and restart the running `busybox` pods. Since a local backend is used in `stash-demo` Restic, sidecar container will mount the corresponding persistent volume.

```console
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS        RESTARTS   AGE
stash-demo-788ffcf9c6-6t6lj   0/1       Terminating   0          3m
stash-demo-79554ff97b-wsdx2   2/2       Running       0          49s
```

```yaml
$ kubectl get deployment stash-demo -o yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    restic.appscode.com/last-applied-configuration: |
      {"kind":"Restic","apiVersion":"stash.appscode.com/v1alpha1","metadata":{"name":"stash-demo","namespace":"default","selfLink":"/apis/stash.appscode.com/v1alpha1/namespaces/default/restics/stash-demo","uid":"d8768901-d8b9-11e7-be92-0800277f19c0","resourceVersion":"27379","creationTimestamp":"2017-12-04T06:10:37Z"},"spec":{"selector":{"matchLabels":{"app":"stash-demo"}},"fileGroups":[{"path":"/source/data","retentionPolicyName":"keep-last-5"}],"backend":{"storageSecretName":"stash-demo","local":{"volumeSource":{"hostPath":{"path":"/data/stash-test/restic-repo"}},"path":"/safe/data"}},"schedule":"@every 1m","volumeMounts":[{"name":"source-data","mountPath":"/source/data"}],"resources":{},"retentionPolicies":[{"name":"keep-last-5","keepLast":5,"prune":true}]},"status":{}}
    restic.appscode.com/tag: canary
  creationTimestamp: 2017-12-04T06:08:55Z
  generation: 2
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
  resourceVersion: "27401"
  selfLink: /apis/extensions/v1beta1/namespaces/default/deployments/stash-demo
  uid: 9c2bf209-d8b9-11e7-be92-0800277f19c0
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
      - args:
        - backup
        - --restic-name=stash-demo
        - --workload-kind=Deployment
        - --workload-name=stash-demo
        - --run-via-cron=true
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
        image: appscode/stash:0.7.0-rc.3
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
      - hostPath:
          path: /data/stash-test/restic-repo
          type: ""
        name: stash-local
status:
  conditions:
  - lastTransitionTime: 2017-12-04T06:10:37Z
    lastUpdateTime: 2017-12-04T06:10:37Z
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: 2017-12-04T06:08:55Z
    lastUpdateTime: 2017-12-04T06:10:37Z
    message: ReplicaSet "stash-demo-79554ff97b" is progressing.
    reason: ReplicaSetUpdated
    status: "True"
    type: Progressing
  observedGeneration: 2
  replicas: 2
  unavailableReplicas: 2
  updatedReplicas: 1
```

Now, wait until the deployment is in running state. Once the pod with sidecar is in running state, it will create a `Repository` CRD for this deployment with name `deployment.stash-demo`. Check the `Repository` CRD object has been created successfully by,

```console
$ kubectl get repository -l workload-name=stash-demo
NAME                    AGE
deployment.stash-demo   3m
```
`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup is taking successfully by,

```console 
$ kubectl get snapshots -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-c1014ca6   10s
```

Here, `deployment.stash-demo-c1014ca6` represents the name of the successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

 You can also check the `status.backupCount` of `deployment.stash-demo` Repository CRD to see number of successful backup taken in this repository.

```console
$ kubectl get repository deployment.stash-demo -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  clusterName: ""
  creationTimestamp: 2018-03-29T08:28:10Z
  generation: 0
  labels:
    restic: stash-demo
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo
  namespace: default
  resourceVersion: "999"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/repositories/deployment.stash-demo
  uid: 1d814fdb-332b-11e8-94e6-08002792cb23
spec:
  backend:
    local:
      hostPath:
        path: /data/stash-test/restic-repo
      mountPath: /safe/data
    storageSecretName: local-secret
  backupPath: deployment/stash-demo
status:
  backupCount: 1
  firstBackupTime: 2018-03-29T08:29:10Z
  lastBackupDuration: 2.105757874s
  lastBackupTime: 2018-03-29T08:29:10Z
```

## Disable Backup
To stop Restic from taking backup, you can do following things:

* Set `spec.paused: true` in Restic `yaml` and then apply the update. This means:

  - Paused Restic CRDs will not applied to newly created wrokloads.
  - Stash sidecar containers will not be removed from existing workloads but the sidecar will stop taking backup.

```command
$ kubectl patch restic stash-demo --type="merge" --patch='{"spec": {"paused": true}}'
restic "stash-demo" patched
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
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
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

* Delete the Restic CRD. Stash operator will remove the sidecar container from all matching workloads.

```commands
$ kubectl delete restic stash-demo
restic "stash-demo" deleted
```
* Change the labels of a workload. Stash operator will remove sidecar container from that workload. This way you can selectively stop backup of a Deployment, ReplicaSet etc.

### Resume Backup
You can resume Restic to backup by setting `spec.paused: false` in Restic `yaml` and applying the update or you can patch Restic using,
```command
$ kubectl patch restic stash-demo --type="merge" --patch='{"spec": {"paused": false}}'
```


## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:
```console
$ kubectl delete deployment stash-demo
$ kubectl delete secret stash-demo
$ kubectl delete restic stash-demo
$ kubectl delete repository deployment.stash-demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

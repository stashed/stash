# Using Stash with Google Kubernetes Engine (GKE)

This tutorial will show you how to use Stash to **backup** and **restore** a Kubernetes deployment in [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/). Here, we are going to backup the `/source/data` folder of a busybox pod into  [GCS bucket](/docs/guides/backends.md#google-cloud-storage-gcs). Then, we will show how to recover this data into a `gcePersistentDisk` and `PersistentVolumeClaim`. We will also re-deploy deployment using this recovered volume.

## Before You Begin

At first, you need to have a Kubernetes cluster in Google Cloud Platform. If you don't already have a cluster, you can create one from [here](https://console.cloud.google.com/kubernetes). Now, install Stash in your cluster following the steps [here](/docs/setup/install.md).

You should have understanding the following Stash concepts:

- [Restic](/docs/concepts/crds/restic.md)
- [Repository](/docs/concepts/crds/repository.md)
- [Recovery](/docs/concepts/crds/recovery.md)

Then, you will need to have a [GCS Bucket](https://console.cloud.google.com/storage) and [GCE persistent disk](https://console.cloud.google.com/compute/disks). GCE persistent disk must be in the same GCE project and zone as the cluster.

## Backup

First, deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as a source volume for demonstration purpose.

```console
$ kubectl apply -f ./busybox.yaml
deployment "stash-demo" created
```

Definition of `busybox` deployment,

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
      - gitRepo:
          repository: https://github.com/appscode/stash-data.git
        name: source-data
```

Run the following command to confirm that `busybox` pods are running.

```console
$ kubectl get pods -l app=stash-demo
NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-b66b9cdfd-8s98d   1/1       Running   0          6m
```

You can check that the `/source/data/` directory of pod is populated with data from the volume source using this command,

```console
$ kubectl exec stash-demo-b66b9cdfd-8s98d -- ls -R /source/data/
/source/data/:
stash-data

/source/data/stash-data:
Eureka-by-EdgarAllanPoe.txt
LICENSE
README.md
```

Now, let’s backup the directory into a [GCS bucket](/docs/guides/backends.md#google-cloud-storage-gcs),

At first, we need to create a secret for `Restic` crd. Create a secret for `Restic` using following command,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-project-id>' > GOOGLE_PROJECT_ID
$ cat downloaded-sa-json.key > GOOGLE_SERVICE_ACCOUNT_JSON_KEY
$ kubectl create secret generic gcs-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./GOOGLE_PROJECT_ID \
    --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret "gcs-secret" created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret gcs-secret -o yaml
```

```yaml
apiVersion: v1
data:
  GOOGLE_PROJECT_ID: <base64 encoded google project id>
  GOOGLE_SERVICE_ACCOUNT_JSON_KEY: <base64 encoded google service account json key>
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2018-04-11T12:57:05Z
  name: gcs-secret
  namespace: default
  resourceVersion: "7113"
  selfLink: /api/v1/namespaces/default/secrets/gcs-secret
  uid: d5e70521-3d87-11e8-a5b9-42010a800002
type: Opaque
```

Now, we can create `Restic` crd. This will create a repository `stash-backup-repo` in GCS bucket and start taking periodic backup of `/source/data/` folder.

```console
$ kubectl apply -f ./gcs-restic.yaml
restic "gcs-restic" created
```

Definition of `Restic` crd for GCS backend,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: gcs-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: demo
    storageSecretName: gcs-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

If everything goes well, a `Repository` crd with name `deployment.stash-demo` will be created for the respective repository in GCS backend. To verify, run the following command,

```console
$ kubectl get repository deployment.stash-demo
NAME                    AGE
deployment.stash-demo   1m
```

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup is taking successfully by,

```console 
$ kubectl get snapshots -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-c1014ca6   10s
```

Here, `deployment.stash-demo-c1014ca6` represents the name of the successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

## Recover to GCE Persistent Disk
Now, we will recover the backed up data into GCE persistent disk. First create a GCE disk named `stash-recovered` from [Google cloud console](https://console.cloud.google.com/compute/disks). Then create `Recovery` crd,

```console
$ kubectl apply -f ./gcs-recovery.yaml
recovery "gcs-recovery" created
```

Definition of `Recovery` crd should look like below:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: gcs-recovery
  namespace: default
spec:
  repository: deployment.stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    gcePersistentDisk:
        pdName: stash-recovered
        fsType: ext4
```

Wait until `Recovery` job completed its task. To verify that recovery completed successfully run,

```console
$ kubectl get recovery -o yaml
```

```yaml
apiVersion: v1
items:
- apiVersion: stash.appscode.com/v1alpha1
  kind: Recovery
  metadata:
    annotations:
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"name":"gcs-recovery","namespace":"default"},"spec":{"repository":"deployment.stash-demo","paths":["/source/data"],"recoveredVolumes":[{"mountPath":"/source/data","gcePersistentDisk":{"pdName":"stash-recovered","fsType":"ext4"}}]}}
    clusterName: ""
    creationTimestamp: 2018-04-12T04:54:46Z
    generation: 0
    name: gcs-recovery
    namespace: default
    resourceVersion: "7388"
    selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/recoveries/gcs-recovery
    uid: 9f886069-3e0d-11e8-951b-42010a80002e
  spec:
    repository: deployment.stash-demo
    paths:
    - /source/data
    recoveredVolumes:
    - gcePersistentDisk:
        fsType: ext4
        pdName: stash-recovered
      mountPath: /source/data
  status:
    phase: Succeeded
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""

```

`status.phase: Succeeded` indicates that the recovery was successful.

Now, let's re-deploy the `busybox` deployment using this recovered volume. First, delete old deployment and recovery job.

```console
$ kubectl delete deployment stash-demo
deployment "stash-demo" deleted

$ kubectl delete recovery gcs-recovery
recovery "gcs-recovery" deleted
```

Now, mount the recovered volume in `busybox` deployment instead of `gitRepo` we had mounted before then re-deploy it.

```console
$ kubectl apply -f ./busybox.yaml
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
        gcePersistentDisk:
          pdName: stash-recovered
          fsType: ext4
```

Get the pod of new deployment,

```console
$ kubectl get pods -l app=stash-demo
NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-857995799-gpml9   1/1       Running   0          34s
```

Check the backed up data is restored in `/source/data/` directory of `busybox` pod.

```console
$ kubectl exec stash-demo-857995799-gpml9 -- ls -R /source/data/
/source/data/:
lost+found
stash-data

/source/data/lost+found:

/source/data/stash-data:
Eureka-by-EdgarAllanPoe.txt
LICENSE
README.md

```

## Recover to `PersistentVolumeClaim`

At first, delete `Restic` crd so that it does not lock the restic repository while we are trying to recover from it.

```console
$ kubectl delete restic rook-restic
restic "rook-restic" deleted
```

Now, create a `PersistentVolumeClaim`,

```console
$ kubectl apply -f ./gcs-pvc.yaml
persistentvolumeclaim "stash-recovered" created
```

Definition of `PersistentVolumeClaim` should look like below:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: stash-recovered
  labels:
    app: stash-demo
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

Check cluster has provisioned the requested claim,

```console
$ kubectl get pvc -l app=stash-demo
NAME              STATUS    VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
stash-recovered   Bound     pvc-57bec6e5-3e11-11e8-951b-42010a80002e   2Gi        RWO            standard       1m
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-57bec6e5-3e11-11e8-951b-42010a80002e`.

Now, create a `Recovery` to recover backed up data in this PVC.

```console
$ kubectl apply -f ./gcs-recovery-pvc.yaml
recovery "gcs-recovery" created
```

Definition of `Recovery` to recover in `PersistentVolumeClaim`,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: gcs-recovery
  namespace: default
spec:
  repository: deployment.stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    persistentVolumeClaim:
      claimName: stash-recovered
```

Wait until `Recovery` job completed its task. To verify that recovery completed successfully run,

```console
$ kubectl get recovery gcs-recovery -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
     {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"name":"gcs-recovery","namespace":"default"},"spec":{"repository":"deployment.stash-demo","paths":["/source/data"],"recoveredVolumes":[{"mountPath":"/source/data","persistentVolumeClaim":{"claimName":"stash-recovered"}}]}}
  clusterName: ""
  creationTimestamp: 2018-04-12T05:26:03Z
  generation: 0
  name: gcs-recovery
  namespace: default
  resourceVersion: "9344"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/recoveries/gcs-recovery
  uid: fe0eb3b9-3e11-11e8-951b-42010a80002e
spec:
  repository: deployment.stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    persistentVolumeClaim:
      claimName: stash-recovered
status:
  phase: Succeeded
```

Now, let's re-deploy the `busybox` deployment using this recovered PVC. First, delete old deployment and recovery job.

```console
$ kubectl delete deployment stash-demo
deployment "stash-demo" deleted

$ kubectl delete recovery gcs-recovery
recovery "gcs-recovery" deleted
```

Now, mount the recovered `PersistentVolumeClaim` in `busybox` deployment instead of `gitRepo` we had mounted before then re-deploy it,

```console
$ kubectl apply -f ./busybox.yaml
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
        persistentVolumeClaim:
          claimName: stash-recovered
```

Get the pod of new deployment,

```console
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-559845c5db-8cd4w   1/1       Running   0          33s
```

Check the backed up data is restored in `/source/data/` directory of `busybox` pod.

```console
$ kubectl exec stash-demo-5bc57fbcfb-mfrfp -- ls -R /source/data/
/source/data/:
lost+found
stash-data

/source/data/lost+found:

/source/data/stash-data:
Eureka-by-EdgarAllanPoe.txt
LICENSE
README.md
```


## Cleanup

```console
$ kubectl delete pvc stash-recovered
$ kubectl delete deployment stash-demo
$ kubectl delete repository deployment.stash-demo
```

Delete the disk created here from Google Cloud console. Uninstall Stash following the instructions [here](/docs/setup/uninstall.md).

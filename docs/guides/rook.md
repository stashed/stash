# Using Stash with Rook Storage Service

This tutorial will show you how to use Stash to **backup** and **restore** a Kubernetes volume in [Rook](https://rook.io/) storage service. Here, we are going to backup the `/source/data` folder of a busybox pod into [AWS S3](/docs/guides/backends.md#aws-s3) compatible [Rook Object Storage](https://rook.io/docs/rook/master/object.html). Then, we will show how to recover this data into a `PersistentVolumeClaim` of [Rook Block Storage](https://rook.io/docs/rook/master/block.html). We will also re-deploy deployment using this recovered volume.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

You should have understanding the following Stash concepts:

- [Restic](/docs/concepts/crds/restic.md)
- [Repository](/docs/concepts/crds/repository.md)
- [Recovery](/docs/concepts/crds/recovery.md)

Then, you will need to have a [Rook Storage Service](https://rook.io) with [Object Storage](https://rook.io/docs/rook/master/object.html) and [Block Storage](https://rook.io/docs/rook/master/block.html) configured. If you do not already have a **Rook Storage Service** configured, you can create one by following this [quickstart guide](https://rook.io/docs/rook/master/quickstart.html).

## Backup

First, deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as a source volume for demonstration purpose.

```console
$ kubectl apply -f ./busybox.yaml
deployment "stash-demo" created
```

Definition of `busybox` deployment:

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
stash-demo-b66b9cdfd-j7rb5   1/1       Running   0          49s
```

You can check that the `/source/data/` directory of pod is populated with data from the volume source using this command,

```console
$  kubectl exec stash-demo-b66b9cdfd-j7rb5 -- ls -R /source/data/
/source/data/:
stash-data

/source/data/stash-data:
Eureka-by-EdgarAllanPoe.txt
LICENSE
README.md
```

Now, let’s backup the directory into a [AWS S3](/docs/guides/backends.md#aws-s3) compatible Rook Object Storage.

At first, we need to create a secret for `Restic` crd. Create secret for `Restic` using following command,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-rook-access-key-id-here>' > AWS_ACCESS_KEY_ID
$ echo -n '<your-rook-secret-access-key-here>' > AWS_SECRET_ACCESS_KEY
$ kubectl create secret generic rook-restic-secret \
      --from-file=./RESTIC_PASSWORD \
      --from-file=./AWS_ACCESS_KEY_ID \
      --from-file=./AWS_SECRET_ACCESS_KEY
secret "rook-restic-secret" created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret rook-restic-secret -o yaml
```

```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: <base64 encoded rook access key>
  AWS_SECRET_ACCESS_KEY: <base64 encoded rook secret key>
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2018-04-12T10:32:14Z
  name: rook-restic-secret
  namespace: default
  resourceVersion: "2414"
  selfLink: /api/v1/namespaces/default/secrets/rook-restic-secret
  uid: c454391b-3e3c-11e8-a7b6-080027672508
type: Opaque
```

Now, we can create `Restic` crd. This will create a repository `stash-backup-repo` in **Rook Object Storage** bucket and start taking periodic backup of `/source/data/` folder.

```console
$ kubectl apply -f ./rook-restic.yaml
restic "rook-restic" created
```

Definition of `Restic` crd for Rook Object Storage backend,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: rook-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo # Must match with the label of busybox pod we have created before.
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    s3:
      endpoint: 'http://rook-ceph-rgw-my-store.rook' # Use your own rook object storage end point.
      bucket: stash-backup  # Give a name of the bucket where you want to backup.
      prefix: demo  # . Path prefix into bucket where repository will be created.(optional).
    storageSecretName: rook-restic-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

If everything goes well, A `Repository` crd with name `deployment.stash-demo` will be created for the respective repository in Rook Object Storage backend. Verify that, `Repository` is created successfully using this command,

```console
$ kubectl get repository deployment.stash-demo
NAME                    AGE
deployment.stash-demo   1m
```

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup is taking successfully by,

```console
$ kubectl get repository deployment.stash-demo -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  clusterName: ""
  creationTimestamp: 2018-04-12T10:44:38Z
  generation: 0
  labels:
    restic: rook-restic
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo
  namespace: default
  resourceVersion: "3436"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/repositories/deployment.stash-demo
  uid: 7fec8b2d-3e3e-11e8-a7b6-080027672508
spec:
  backend:
    s3:
      bucket: stash-backup
      endpoint: http://rook-ceph-rgw-my-store.rook
      prefix: stash-backup/demo/deployment/stash-demo
    storageSecretName: rook-restic-secret
status:
  backupCount: 2
  firstBackupTime: 2018-04-12T10:45:44Z
  lastBackupDuration: 7.766740386s
  lastBackupTime: 2018-04-12T10:46:41Z
```

Look at the `status` field. `backupCount` show number of successful backup taken in this `Repository`.

## Recover to `PersistentVolumeClaim`

**Rook Block Storage** allow the users to mount persistent volume into pod using  `PersistentVolumeClaim`. Here, we will recover our backed up data into a PVC.

At first, delete `Restic` crd so that it does not lock the restic repository while we are trying to recover from it.

```console
$ kubectl delete restic rook-restic
restic "rook-restic" deleted
```

Now, create a `PersistentVolumeClaim` for Rook Block Storage,

```console
$ kubectl apply -f ./rook-pvc.yaml
persistentvolumeclaim "stash-recovered" created
```

Definition of `PersistentVolumeClaim`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: stash-recovered
  labels:
    app: stash-demo
spec:
  storageClassName: rook-block
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
stash-recovered   Bound     pvc-a7aa73fa-3e3f-11e8-a7b6-080027672508   2Gi        RWO            rook-block     36s
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-a7aa73fa-3e3f-11e8-a7b6-080027672508`.

Now, create a `Recovery` to recover backed up data in this PVC.

```console
$ kubectl apply -f ./rook-recovery.yaml
recovery "rook-recovery" created
```

Definition of `Recovery` should look like below:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: rook-recovery
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
$ kubectl get recovery rook-recovery -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
     {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"name":"rook-recovery","namespace":"default"},"spec":{"repository":"deployment.stash-demo","paths":["/source/data"],"recoveredVolumes":[{"mountPath":"/source/data","persistentVolumeClaim":{"claimName":"stash-recovered"}}]}}
  clusterName: ""
  creationTimestamp: 2018-04-12T12:57:54Z
  generation: 0
  name: rook-recovery
  namespace: default
  resourceVersion: "2315"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/recoveries/rook-recovery
  uid: 1dbad356-3e51-11e8-b2bd-080027dbef96
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

$ kubectl delete recovery rook-recovery
recovery "rook-recovery" deleted
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
$  kubectl get pod -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-5bc57fbcfb-b45k8   1/1       Running   0          1m
```

Check the backed up data is restored in `/source/data/` directory of `busybox` pod.

```console
$ kubectl exec stash-demo-5bc57fbcfb-b45k8 -- ls -R /source/data/
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

Uninstall Stash following the instructions [here](/docs/setup/uninstall.md).
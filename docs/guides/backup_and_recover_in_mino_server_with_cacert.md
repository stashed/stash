# Backup and Recovery in TLS secure Minio Server Using Stash

This tutorial will show you how to use [Stash](./../concepts/what-is-stash/overview.md) to backup a Kubernetes `Deployment` in a TLS secure [Minio](https://docs.minio.io/) Server. It will also show you how to recover this backed up data.

## Prerequisites

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster,
you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

Then, you will need a TLS secure [Minio](https://docs.minio.io/) server to store backed up data. You can deploy a TLS secure Minio server in your cluster by following this [guide](./minio_server.md).

You must have understanding the following Stash terms: [Restic](./../concepts/crds/restic.md) and [Recovery](./../concepts/crds/recovery.md).

## Overview

In this tutorial, we are going to backup the `/source/data` folder of a `busybox` pod into a `Minio` backend. Then, we will recover the data to another `HostPath` volume form backed up snapshots.

## Backup

First, deploy the following `busybox` Deployment in your cluster. Here we are using a git repository as a source volume for demonstration purpose.

```console
$  kubectl apply -f ./busybox.yaml
deployment "stash-demo" created
```

YAML for `busybox` deplyment,

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
stash-demo-69d9dd8d76-bz2bz   1/1       Running   0          12s
```

You can check that the `/source/data/` directory of pod is populated with data from the volume source using this command,

```console
$ kubectl exec stash-demo-69d9dd8d76-bz2bz -- ls -R /source/data/
/source/data/:
stash-data

/source/data/stash-data:
Eureka-by-EdgarAllanPoe.txt
LICENSE
README.md
```

Now, let's backup the directory into a Minio server.

At first, we need to create a secret for `Restic` crd. To configure this backend, following secret keys are needed:

| Key                     | Description                                                             |
|-------------------------|-------------------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic`              |
| `AWS_ACCESS_KEY_ID`     | `Required`. Minio access key ID                                         |
| `AWS_SECRET_ACCESS_KEY` | `Required`. Minio secret access key                                     |
| `CA_CERT_DATA`          |`Required`. Root certificate by which Minio server certificate is signed |

Create secret for `Restic` crd,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-minio-access-key-id-here>' > AWS_ACCESS_KEY_ID
$ echo -n '<your-minio-secret-access-key-here>' > AWS_SECRET_ACCESS_KEY
$ cat ./directory/of/root/certificate/ca.crt > CA_CERT_DATA
$ kubectl create secret generic minio-restic-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./AWS_ACCESS_KEY_ID \
    --from-file=./AWS_SECRET_ACCESS_KEY \
    --from-file=./CA_CERT_DATA
secret "minio-restic-secret" created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret minio-restic-secret -o yaml
```

```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: PGVtcnV6Pg==
  AWS_SECRET_ACCESS_KEY: PDEyMzQ1Njc4OTA+
  CA_CERT_DATA: <base64 endoded ca.crt data>
  RESTIC_PASSWORD: ZW1ydXo=
kind: Secret
metadata:
  creationTimestamp: 2018-01-29T11:20:35Z
  name: minio-restic-secret
  namespace: default
  resourceVersion: "7773"
  selfLink: /api/v1/namespaces/default/secrets/minio-restic-secret
  uid: 6d70a2c1-04e6-11e8-b4cd-0800279de528
type: Opaque
```

Now, we can create `Restic` crd. This will create a repository in Minio server and start taking periodic backup of `/source/data/` folder.

```console
$ kubectl apply -f ./minio-restic.yaml
restic "minio-restic" created
```

YAML of `Restic` crd for Minio backend,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: minio-restic
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
      endpoint: 'https://192.168.99.100:32321' # Use your own Minio server address.
      bucket: stash-qa  # Give a name of the bucket where you want to backup.
      prefix: demo  # . Path prefix into bucket where repository will be created.(optional).
    storageSecretName: minio-restic-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

If everything goes well, `Restic` will take backup of the volume periodically with 1 minute interval. You can see if the backup working correctly using this command
,
```console
$ kubectl get restic minio-restic -o yaml
```

Output will be something similar to,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Restic","metadata":{"annotations":{},"name":"minio-restic","namespace":"default"},"spec":{"backend":{"s3":{"bucket":"stash-qa","endpoint":"https://192.168.99.100:32321","prefix":"demo"},"storageSecretName":"minio-restic-secret"},"fileGroups":[{"path":"/source/data","retentionPolicyName":"keep-last-5"}],"retentionPolicies":[{"keepLast":5,"name":"keep-last-5","prune":true}],"schedule":"@every 1m","selector":{"matchLabels":{"app":"stash-demo"}},"volumeMounts":[{"mountPath":"/source/data","name":"source-data"}]}}
  clusterName: ""
  creationTimestamp: 2018-01-30T04:45:09Z
  generation: 0
  name: minio-restic
  namespace: default
  resourceVersion: "4385"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/minio-restic
  uid: 5a0651c3-0578-11e8-9976-08002750604b
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: https://192.168.99.100:32321
      prefix: demo
    storageSecretName: minio-restic-secret
  fileGroups:
  - path: /source/data
    retentionPolicyName: keep-last-5
  retentionPolicies:
  - keepLast: 5
    name: keep-last-5
    prune: true
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
  volumeMounts:
  - mountPath: /source/data
    name: source-data
status:
  backupCount: 13
  firstBackupTime: 2018-01-30T04:46:41Z
  lastBackupDuration: 2.727698782s
  lastBackupTime: 2018-01-30T04:58:41Z
```

Look at the `status` field. `backupCount` show number of successful backup done by the `Restic`.

## Recovery

Now, it is time to recover the backed up data. First, create a  `Recovery` crd.

```console
$ kubectl apply -f ./minio-recovery.yaml
recovery "minio-recovery" created
```

YAML for `Recovery` crd

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: minio-recovery
  namespace: default
spec:
  workload:
    kind: Deployment
    name: stash-demo # Must match with the label of busybox pod we are recoverying.
  backend:
    s3:
      endpoint: 'https://192.168.99.100:32321' # use your own Minio server address
      bucket: stash-qa
      prefix: demo
    storageSecretName: minio-restic-secret # we will use same secret created for Restic crd. You can create new secret for Recovery with same credentials.
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data # where the volume will be mounted
    name: stash-recovered-volume
    hostPath: # volume source, where the recovered data will be stored.
      path: /data/stash-recovered/ # directory in volume source where recovered data will be stored
```

Check whether the recovery was successful,

```console
$ kubectl get recovery minio-recovery -o yaml
```

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"annotations":{},"name":"minio-recovery","namespace":"default"},"spec":{"backend":{"s3":{"bucket":"stash-qa","endpoint":"https://192.168.99.100:32321","prefix":"demo"},"storageSecretName":"minio-restic-secret"},"paths":["/source/data"],"recoveredVolumes":[{"hostPath":{"path":"/data/stash-recovered/"},"mountPath":"/source/data","name":"stash-recovered-volume"}],"workload":{"kind":"Deployment","name":"stash-demo"}}}
  clusterName: ""
  creationTimestamp: 2018-01-30T06:54:18Z
  generation: 0
  name: minio-recovery
  namespace: default
  resourceVersion: "10060"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/recoveries/minio-recovery
  uid: 64c12ff7-058a-11e8-9976-08002750604b
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: https://192.168.99.100:32321
      prefix: demo
    storageSecretName: minio-restic-secret
  paths:
  - /source/data
  recoveredVolumes:
  - hostPath:
      path: /data/stash-recovered/
    mountPath: /source/data
    name: stash-recovered-volume
  workload:
    kind: Deployment
    name: stash-demo
status:
  phase: Succeeded
```

`status.phase: Succeeded` indicates that the recovery was successful. Now, we can check `/data/stash-recovered/` directory of `HostPath` to see the recovered data. If you are using `minikube` for cluster then you can check by,

```console
$ minikube ssh
                         _             _            
            _         _ ( )           ( )           
  ___ ___  (_)  ___  (_)| |/')  _   _ | |_      __  
/' _ ` _ `\| |/' _ `\| || , <  ( ) ( )| '_`\  /'__`\
| ( ) ( ) || || ( ) || || |\`\ | (_) || |_) )(  ___/
(_) (_) (_)(_)(_) (_)(_)(_) (_)`\___/'(_,__/'`\____)

$ sudo su
$ cd /
$ ls -R /data/stash-recovered/
/data/stash-recovered/:
data

/data/stash-recovered/data:
stash-data

/data/stash-recovered/data/stash-data:
Eureka-by-EdgarAllanPoe.txt  LICENSE  README.md
```

You can mount this recovered volume into any pod.

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete deployment stash-demo
$ kubectl delete restic minio-restic
$ kubectl delete recovery minio-recovery
$ kubectl delete secret minio-restic-secret
```
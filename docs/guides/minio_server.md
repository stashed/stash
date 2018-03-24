---
title: Minio | Stash
description: Using Stash with TLS secured Minio Server
menu:
  product_stash_0.7.0-rc.2:
    identifier: minio-stash
    name: Backup to Minio
    parent: guides
    weight: 45
product_name: stash
menu_name: product_stash_0.7.0-rc.2
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Using Stash with TLS secured Minio Server

Minio is an open source object storage server compatible with Amazon S3 cloud storage service. You can deploy Minio server in docker container locally, in a kubernetes cluster, Microsoft Azure, GCP etc. You can find a guide for Minio server [here](https://docs.minio.io/). This tutorial will show you how to use [Stash](/docs/concepts/what-is-stash/overview.md) to backup a Kubernetes `Deployment` in a TLS secure [Minio](https://docs.minio.io/) Server. It will also show you how to recover this backed up data.

## Prerequisites

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube). Now, install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

You should have understanding the following Stash terms:

- [Restic](/docs/concepts/crds/restic.md)
- [Recovery](/docs/concepts/crds/recovery.md)

Then, you will need a TLS secure [Minio](https://docs.minio.io/) server to store backed up data. You can deploy a TLS secure Minio server in your cluster by following the steps below:

### Create self-signed SSl certificate

A Certificate is used to verify the identity of server or client. Usually, a certificate issued by trusted third party is used to verify identity. We can also use a self-signed certificate. In this tutorial, we will use a self-signed certificate to verify the identity of Minio server.

You can generate self-signed certificate easily with our [onessl](https://github.com/appscode/onessl) tool.

Here is an example how we can generate a self-signed certificate using `onessl` tool.

First install onessl by,

```console
$ curl -fsSL -o onessl https://github.com/appscode/onessl/releases/download/0.1.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/
```

Now generate CA's root certificate,

```console
$ onessl create ca-cert
```

This will create two files `ca.crt` and `ca.key`.

Now, generate  certificate for server,

```console
$ onessl create server-cert --domains minio-service.default.svc
```

This will generate two files `server.crt` and `server.key`.

Minio server will start TLS secure service if it find `public.crt` and `private.key` files in `/root/.minio/certs/` directory of the docker container. The `public.crt` file is concatenation of `server.crt` and `ca.crt` where `private.key` file is only the `server.key` file.

Let's generate `public.crt` and `private.key` file,

```console
$ cat {server.crt,ca.crt} > public.crt
$ cat server.key > private.key
```

Be sure about the order of `server.crt`  and `ca.crt`. The order will be `server's certificate`, any `intermediate certificates` and finally the `CA's root certificate`. The intermediate certificates are required if the server certificate is created using a certificate which is not the root certificate but signed by the root certificate. [onessl](https://github.com/appscode/onessl) use root certificate by default to generate server certificate if no certificate path is specified by `--cert-dir` flag. Hence, the intermediate certificates are not required here.

We will create a kubernetes secret with this `public.crt` and `private.key` files and mount the secret to `/root/.minio/certs/` directory of minio container.

### Create Secret

Now, let's create a secret from `public.crt` and `private.key` files,

```console
$ kubectl create secret generic minio-server-secret \
                              --from-file=./public.crt \
                              --from-file=./private.key
secret "minio-server-secret" created

$ kubectl label secret minio-server-secret app=minio -n default
```

Now, verify that the secret is created successfully

```console
$ kubectl get secret minio-server-secret -o yaml
```

If secret is created successfully then you will see output like this,

```yaml
apiVersion: v1
data:
  private.key: <base64 encoded private.key data>
  public.crt: <base64 encoded public.key data>
kind: Secret
metadata:
  creationTimestamp: 2018-01-26T12:02:09Z
  name: minio-server-secret
  namespace: default
  resourceVersion: "40701"
  selfLink: /api/v1/namespaces/default/secrets/minio-server-secret
  uid: bc57add7-0290-11e8-9a26-080027b344c9
  labels:
    app: minio
type: Opaque
```

### Create Persistent Volume Claim

Minio server needs a Persistent Volume to store data. Let's create a `Persistent Volume Claim` to request Persistent Volume from the cluster.

```console
$ kubectl apply -f ./minio-pvc.yaml
persistentvolumeclaim "minio-pvc" created
```

YAML for PersistentVolumeClaim,

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  # This name uniquely identifies the PVC. Will be used in minio deployment.
  name: minio-pvc
  labels:
    app: minio
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    # This is the request for storage. Should be available in the cluster.
    requests:
      storage: 2Gi
```

### Create Deployment

Minio deployment creates pod where the Minio server will run. Let's create a deployment for minio server by,

```console
$ kubectl apply -f ./minio-deployment.yaml
deployment "minio-deployment" created
```

YAML for minio-deployment

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  # This name uniquely identifies the Deployment
  name: minio-deployment
  labels:
    app: minio
spec:
  strategy:
    type: Recreate # If pod fail, we want to recreate pod rather than restarting it.
  template:
    metadata:
      labels:
        # Label is used as a selector in the service.
        app: minio-server
    spec:
      volumes:
      # Refer to the PVC have created earlier
      - name: storage
        persistentVolumeClaim:
          # Name of the PVC created earlier
          claimName: minio-pvc
      # Refer to minio-server-secret we have created earlier
      - name: minio-server-secret
        secret:
          secretName: minio-server-secret
      containers:
      - name: minio
        # Pulls the default Minio image from Docker Hub
        image: minio/minio
        args:
        - server
        - --address
        - ":443"
        - /storage
        env:
        # Minio access key and secret key
        - name: MINIO_ACCESS_KEY
          value: "<your minio access key(any string)>"
        - name: MINIO_SECRET_KEY
          value: "<your minio secret key(any string)>"
        ports:
        - containerPort: 443
          # This ensures containers are allocated on separate hosts. Remove hostPort to allow multiple Minio containers on one host
          hostPort: 443
        # Mount the volumes into the pod
        volumeMounts:
        - name: storage # must match the volume name, above
          mountPath: "/storage"
        - name: minio-server-secret
          mountPath: "/root/.minio/certs/" # directory where the certificates will be mounted
```

### Create Service

Now, the final touch. Minio server is running in the cluster. Let's create a service so that other pods can access the server.

```console
$ kubectl apply -f ./minio-service.yaml
service "minio-service" created
```

YAML for minio-service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: minio-service
  labels:
    app: minio
spec:
  type: LoadBalancer
  ports:
    - port: 443
      targetPort: 443
      protocol: TCP
  selector:
    app: minio-server # must match with the label used in the deployment
```

Verify that the service is created successfully,

```console
$ kubectl get service minio-service
NAME            TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
minio-service   LoadBalancer   10.106.121.137   <pending>     443:30722/TCP   49s
```

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
      endpoint: 'https://minio-service.default.svc' # Use your own Minio server address.
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
      {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Restic","metadata":{"annotations":{},"name":"minio-restic","namespace":"default"},"spec":{"backend":{"s3":{"bucket":"stash-qa","endpoint":"https://minio-service.default.svc","prefix":"demo"},"storageSecretName":"minio-restic-secret"},"fileGroups":[{"path":"/source/data","retentionPolicyName":"keep-last-5"}],"retentionPolicies":[{"keepLast":5,"name":"keep-last-5","prune":true}],"schedule":"@every 1m","selector":{"matchLabels":{"app":"stash-demo"}},"volumeMounts":[{"mountPath":"/source/data","name":"source-data"}]}}
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
      endpoint: https://minio-service.default.svc
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
      endpoint: 'https://minio-service.default.svc' # use your own Minio server address
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
      {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"annotations":{},"name":"minio-recovery","namespace":"default"},"spec":{"backend":{"s3":{"bucket":"stash-qa","endpoint":"https://minio-service.default.svc","prefix":"demo"},"storageSecretName":"minio-restic-secret"},"paths":["/source/data"],"recoveredVolumes":[{"hostPath":{"path":"/data/stash-recovered/"},"mountPath":"/source/data","name":"stash-recovered-volume"}],"workload":{"kind":"Deployment","name":"stash-demo"}}}
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
      endpoint: https://minio-service.default.svc
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

To cleanup the minio server, run:

```console
$ kubectl delete deployment minio-deployment
$ kubectl delete service minio-service
$ kubectl delete pvc minio-pvc
$ kubectl delete secret minio-server-secret
```

---
title: Minio | Stash
description: Using Stash with TLS secured Minio Server
menu:
  product_stash_0.8.3:
    identifier: platforms-minio
    name: Minio
    parent: platforms
    weight: 40
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Using Stash with TLS secured Minio Server

[Minio](https://minio.io/) is an open source object storage server compatible with AWS S3 cloud storage service. This tutorial will show you how to use Stash to **backup** and **restore** a volume with a Minio backend. Here, we are going to backup the `/source/data` folder of a busybox pod into a Minio bucket. Then, we are going to show how to recover this data into a `PersistentVolumeClaim(PVC)`. We are going to also re-deploy deployment using this recovered volume.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the `kubectl` command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

- Install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

- You should be familiar with the following Stash concepts:
  - [Restic](/docs/concepts/crds/restic.md)
  - [Repository](/docs/concepts/crds/repository.md)
  - [Recovery](/docs/concepts/crds/recovery.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

- You will need a TLS secured [Minio](https://docs.minio.io/) server to store backed up data. If you already do not have a Minio server running, deploy one following the tutorial from [here](https://github.com/appscode/third-party-tools/blob/master/storage/minio/README.md). For this tutorial, we have deployed Minio server in `storage` namespace and it is accessible through `minio.storage.svc` dns.

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/platforms/minio](/docs/examples/platforms/minio) directory of [appscode/stash](https://github.com/stashed/stash) repository.

## Backup

In order to take backup, we need some sample data. Stash has some sample data in [stash-data](https://github.com/stashed/stash-data) repository. As [gitRepo](https://kubernetes.io/docs/concepts/storage/volumes/#gitrepo) volume has been deprecated, we are not going to use this repository as volume directly. Instead, we are going to create a [configMap](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) from the stash-data repository and use that ConfigMap as data source.

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
$ kubectl apply -f ./docs/examples/platforms/minio/deployment.yaml
deployment.apps/stash-demo created
```

Now, wait for deployment's pod to go in `Running` state.

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-7ccd56bf5d-n24vl   1/1     Running   0          16s
```

You can check that the `/source/data/` directory of this pod is populated with data from the `stash-sample-data` ConfigMap using this command,

```console
$ kubectl exec -n demo stash-demo-7ccd56bf5d-n24vl -- ls -R /source/data
/source/data:
LICENSE
README.md
```

Now, we are ready to backup `/source/data` directory into a Minio bucket.

**Create Secret:**

At first, we need to create a secret for `Restic` crd. To configure this backend, the following secret keys are needed:

|           Key           |                                              Description                                              |
| ----------------------- | ----------------------------------------------------------------------------------------------------- |
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic`                                            |
| `AWS_ACCESS_KEY_ID`     | `Required`. Minio access key                                                                          |
| `AWS_SECRET_ACCESS_KEY` | `Required`. Minio secret access key                                                                   |
| `CA_CERT_DATA`          | `Required` for TLS secured Minio server. Root certificate by which Minio server certificate is signed |

Create the secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-minio-access-key-id-here>' > AWS_ACCESS_KEY_ID
$ echo -n '<your-minio-secret-access-key-here>' > AWS_SECRET_ACCESS_KEY
$ cat ./directory/of/root/certificate/ca.crt > CA_CERT_DATA
$ kubectl create secret generic -n demo minio-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./AWS_ACCESS_KEY_ID \
    --from-file=./AWS_SECRET_ACCESS_KEY \
    --from-file=./CA_CERT_DATA
secret/minio-secret created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret -n demo minio-secret -o yaml
```

```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: bXktYWNjZXNzLWtleQ==
  AWS_SECRET_ACCESS_KEY: bXktc2NyZXQta2V5
  CA_CERT_DATA: <base64 endoded ca.crt data>
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2018-12-05T11:42:48Z
  name: minio-secret
  namespace: demo
  resourceVersion: "36660"
  selfLink: /api/v1/namespaces/demo/secrets/minio-secret
  uid: e3a2f905-f882-11e8-9a81-0800272171a4
type: Opaque
```

**Create Restic:**

Now, we are going to create `Restic` crd to take backup `/source/data` directory of `stash-demo` deployment. This will create a repository in the Minio bucket specified by `s3.bucket` field and start taking periodic backup of `/source/data` directory.

```console
$ kubectl apply -f ./docs/examples/platforms/minio/restic.yaml
restic.stash.appscode.com/minio-restic created
```

Below, the YAML for Restic crd we have created above,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: minio-restic
  namespace: demo
spec:
  selector:
    matchLabels:
      app: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    s3:
      endpoint: 'https://minio.storage.svc'
      bucket: stash-repo
      prefix: demo
    storageSecretName: minio-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

If everything goes well, Stash will inject a sidecar container into the `stash-demo` deployment to take periodic backup. Let's check that sidecar has been injected successfully,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-57656f6d74-hmc9z   2/2     Running   0          46s
```

Look at the pod. It now has 2 containers. If you view the resource definition of this pod, you will see that there is a container named `stash` which running `backup` command.

**Verify Backup:**

Stash will create a `Repository` crd with name `deployment.stash-demo` for the respective repository in Minio backend at first backup schedule. To verify, run the following command,

```console
$ kubectl get repository deployment.stash-demo -n demo
NAME                    BACKUPCOUNT   LASTSUCCESSFULBACKUP   AGE
deployment.stash-demo   1             14s                    1m
```

Here, `BACKUPCOUNT` field indicates number of backup snapshot has taken in this repository.

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup snapshots has been created successfully by,

```console
$ kubectl get snapshots -n demo -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-c588c67c   4m3s
deployment.stash-demo-7dc0c17d   3m3s
deployment.stash-demo-21228047   2m3s
deployment.stash-demo-15873428   63s
deployment.stash-demo-a29263b1   3s
```

Here, we can see 5 last successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

If you navigate to `<bucket name>/demo/deployment/stash-demo` directory in your Minio Web UI. You will see, a repository has been created there.

<p align="center">
  <img alt="Repository in Minio Backend",  height="350px" src="/docs/images/platforms/minio/minio-repository.png">
</p>

To view the snapshot files, navigate to `snapshots` directory of the repository,

<p align="center">
  <img alt="Snapshot in Minio Bucket" height="350px" src="/docs/images/platforms/minio/minio-snapshots.png">
</p>

> Stash keeps all backup data encrypted. So, snapshot files in the bucket will not contain any meaningful data until they are decrypted.

## Recovery

Now, consider that we have lost our workload as well as data volume. We want to recover the data into a new volume and re-deploy the workload.

At first, let's delete `Restic` crd, `stash-demo` deployment and `stash-sample-data` ConfigMap.

```console
$ kubectl delete deployment -n demo stash-demo
deployment.extensions "stash-demo" deleted

$ kubectl delete restic -n demo minio-restic
restic.stash.appscode.com "minio-restic" deleted

$ kubectl delete configmap -n demo stash-sample-data
configmap "stash-sample-data" deleted
```

In order to perform recovery, we need `Repository` crd `deployment.stah-demo` and backend secret `minio-secret` to exist.

>In case of cluster disaster, you might lose `Repository` crd and backend secret. In this scenario, you have to create the secret again and `Repository` crd manually. Follow the guide to understand `Repository` crd structure from [here](/docs/concepts/crds/repository.md).

**Create PVC:**

We will recover backed up data into a PVC. At first, we need to know available [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) in our cluster.

```console
$ kubectl get storageclass
NAME                 PROVISIONER                AGE
standard (default)   k8s.io/minikube-hostpath   8h
```

Now, let's create a `PersistentVolumeClaim` where our recovered data will be stored.

```console
$ kubectl apply -f ./docs/examples/platforms/minio/pvc.yaml
persistentvolumeclaim/stash-recovered created
```

Below the YAML for `PersistentVolumeClaim` we have created above,

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: stash-recovered
  namespace: demo
  labels:
    app: stash-demo
spec:
  storageClassName: standard
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 50Mi
```

Check that if cluster has provisioned the requested claim,

```console
$ kubectl get pvc -n demo -l app=stash-demo
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
stash-recovered   Bound    pvc-3d3b6a58-f886-11e8-9a81-0800272171a4   50Mi       RWO            standard       13s
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-3d3b6a58-f886-11e8-9a81-0800272171a4`.

**Create Recovery:**

Now, we have to create a `Recovery` crd to recover backed up data into this PVC.

```console
$ kubectl apply -f ./docs/examples/platforms/minio/recovery.yaml
recovery.stash.appscode.com/minio-recovery created
```

Below, the YAML for `Recovery` crd we have created above.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: minio-recovery
  namespace: demo
spec:
  repository:
    name: deployment.stash-demo
    namespace: demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    persistentVolumeClaim:
      claimName: stash-recovered
```

Wait until `Recovery` job completes its task. To verify that recovery has completed successfully run,

```console
$ kubectl get recovery -n demo minio-recovery
NAME            REPOSITORYNAMESPACE   REPOSITORYNAME          SNAPSHOT   PHASE       AGE
minio-recovery   demo                  deployment.stash-demo              Succeeded   26s
```

Here, `PHASE` `Succeeded` indicates that our recovery has been completed successfully. Backup data has been restored in `stash-recovered` PVC. Now, we are ready to use this PVC to re-deploy the workload.

If you are using Kubernetes version older than v1.11.0 then run following command and check `status.phase` field to see whether the recovery succeeded or failed.

```console
$ kubectl get recovery -n demo minio-recovery -o yaml
```

**Re-deploy Workload:**

We have successfully restored backed up data into `stash-recovered` PVC. Now, we are going to re-deploy our previous deployment `stash-demo`. This time, we are going to mount the `stash-recovered` PVC as `source-data` volume instead of ConfigMap `stash-sample-data`.

Below, the YAML for `stash-demo` deployment with `stash-recovered` PVC as `source-data` volume.

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
        persistentVolumeClaim:
          claimName: stash-recovered
```

Let's create the deployment,

```console
$ kubectl apply -f ./docs/examples/platforms/minio/recovered-deployment.yaml
deployment.apps/stash-demo created
```

**Verify Recovered Data:**

We have re-deployed `stash-demo` deployment with recovered volume. Now, it is time to verify that the recovered data are present in `/source/data` directory.

Get the pod of new deployment,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-69694789df-wq9vc   1/1     Running   0          14s
```

Run following command to view data of `/source/data` directory of this pod,

```console
$ kubectl exec -n demo stash-demo-69694789df-wq9vc -- ls -R /source/data
/source/data:
LICENSE
README.md
```

So, we can see that the data we had backed up from original deployment are now present in re-deployed deployment.

## Cleanup

To cleanup the resources created by this tutorial, run following commands:

```console
$ kubectl delete recovery -n demo minio-recovery
$ kubectl delete secret -n demo minio-secret
$ kubectl delete deployment -n demo stash-demo
$ kubectl delete pvc -n demo stash-recovered
$ kubectl delete repository -n demo deployment.stash-demo

$ kubectl delete ns demo
```

- To uninstall Stash from your cluster, follow the instructions from [here](/docs/setup/uninstall.md).
- If you have deployed Minio server by following the tutorial we have linked in [Before You Begin](/docs/guides/platforms/minio.md#before-you-begin) section, please clean-up Minio resources by following the [cleanup](https://github.com/appscode/third-party-tools/blob/master/storage/minio/README.md#cleanup) section of that tutorial.
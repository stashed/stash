---
title: EKS | Stash
description: Using Stash in Amazon EKS
menu:
  product_stash_0.8.1:
    identifier: platforms-eks
    name: EKS
    parent: platforms
    weight: 10
product_name: stash
menu_name: product_stash_0.8.1
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Using Stash with Amazon EKS

This tutorial will show you how to use Stash to **backup** and **restore** a volume in [Amazon Elastic Container Service for Kubernetes (EKS)](https://aws.amazon.com/eks/). Here, we are going to backup the `/source/data` folder of a busybox pod into [AWS S3 bucket](https://aws.amazon.com/s3/). Then, we are going to show how to recover this data into a `PersistentVolumeClaim(PVC)`. We are going to also re-deploy deployment using this recovered volume.

## Before You Begin

At first, you need to have a EKS cluster. If you don't already have a cluster, create one from [here](https://aws.amazon.com/eks/). You can use [eksctl](https://github.com/weaveworks/eksctl) command line tool to create EKS cluster easily.

- Install Stash in your cluster following the steps [here](/docs/setup/install.md).

- You should be familiar with the following Stash concepts:
  - [Restic](/docs/concepts/crds/restic.md)
  - [Repository](/docs/concepts/crds/repository.md)
  - [Recovery](/docs/concepts/crds/recovery.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

- You will need a [AWS S3 Bucket](https://aws.amazon.com/s3/) to store the backup snapshots.

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/platforms/eks](/docs/examples/platforms/eks) directory of [appscode/stash](https://github.com/appscode/stash) repository.

## Backup

In order to take backup, we need some sample data. Stash has some sample data in [stash-data](https://github.com/appscode/stash-data) repository. As [gitRepo](https://kubernetes.io/docs/concepts/storage/volumes/#gitrepo) volume has been deprecated, we are not going to use this repository as volume directly. Instead, we are going to create a [configMap](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) from the stash-data repository and use that ConfigMap as data source.

Let's create a ConfigMap from these sample data,

```console
$ kubectl create configmap -n demo stash-sample-data \
	--from-literal=LICENSE="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash-data/master/LICENSE)" \
	--from-literal=README.md="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash-data/master/README.md)"
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
$ kubectl apply -f ./docs/examples/platforms/eks/deployment.yaml
deployment.apps/stash-demo created
```

Now, wait for deployment's pod to go in `Running` state.

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                         READY   STATUS    RESTARTS   AGE
stash-demo-756bf59b5-7tk6q   1/1     Running   0          1m
```

You can check that the `/source/data/` directory of this pod is populated with data from the `stash-sample-data` ConfigMap using this command,

```console
$ kubectl exec -n demo stash-demo-756bf59b5-7tk6q  -- ls -R /source/data
/source/data:
LICENSE
README.md
```

Now, we are ready to backup `/source/data` directory into [AWS S3 Bucket](https://aws.amazon.com/s3/).

**Create Secret:**

At first, we need to create a storage secret that hold the credentials for the backend. To configure this backend, the following secret keys are needed:

|           Key           |                        Description                         |
| ----------------------- | ---------------------------------------------------------- |
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AWS_ACCESS_KEY_ID`     | `Required`. AWS access key ID for bucket                   |
| `AWS_SECRET_ACCESS_KEY` | `Required`. AWS secret access key for bucket               |

Create a the storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-aws-access-key-id-here>' > AWS_ACCESS_KEY_ID
$ echo -n '<your-aws-secret-access-key-here>' > AWS_SECRET_ACCESS_KEY
$ kubectl create secret generic -n demo s3-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./AWS_ACCESS_KEY_ID \
    --from-file=./AWS_SECRET_ACCESS_KEY
secret/s3-secret created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret -n demo s3-secret -o yaml
```

```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: <base64 encoded AWS_ACCESS_KEY_ID>
  AWS_SECRET_ACCESS_KEY: <base64 encoded AWS_SECRET_ACCESS_KEY>
  RESTIC_PASSWORD: Y2hhbmdlaXQK
kind: Secret
metadata:
  creationTimestamp: 2018-11-14T10:10:57Z
  name: s3-secret
  namespace: demo
  resourceVersion: "16345"
  selfLink: /api/v1/namespaces/demo/secrets/s3-secret
  uid: 94842507-e7f5-11e8-a7b0-029842a88ece
type: Opaque
```

**Create Restic:**

Now, we are going to create `Restic` crd to take backup `/source/data` directory of `stash-demo` deployment. This will create a repository in the S3 bucket specified in `s3.bucket` field and start taking periodic backup of `/source/data` directory.

```console
$ kubectl apply -f ./docs/examples/platforms/eks/restic.yaml
restic.stash.appscode.com/s3-restic created
```

Below, the YAML for Restic crd we have created above,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: s3-restic
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
      endpoint: 's3.amazonaws.com'
      bucket: stash-qa
      prefix: demo
    storageSecretName: s3-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

If everything goes well, Stash will inject a sidecar container into the `stash-demo` deployment to take periodic backup. Let's check sidecar has been injected successfully,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-646c854778-t4d72   2/2     Running   0          1m
```

Look at the pod. It now has 2 containers. If you view the resource definition of this pod, you will see there is a container named `stash` which running `backup` command.

**Verify Backup:**

Stash will create a `Repository` crd with name `deployment.stash-demo` for the respective repository in S3 backend at first backup schedule. To verify, run the following command,

```console
$  kubectl get repository deployment.stash-demo -n demo
NAME                    CREATED AT
deployment.stash-demo   5m
```

If you view the YAML of this repository, you will see a count of backup snapshots taken in the backend at `status.backupCount` field.

```yaml
$ kubectl get repository deployment.stash-demo -n demo -o yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  clusterName: ""
  creationTimestamp: 2018-11-14T11:41:39Z
  finalizers:
  - stash
  generation: 1
  labels:
    restic: s3-restic
    workload-kind: Deployment
    workload-name: stash-demo
  name: deployment.stash-demo
  namespace: demo
  resourceVersion: "29810"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/demo/repositories/deployment.stash-demo
  uid: 3fea14ca-e802-11e8-9c28-06995e414a88
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: s3.amazonaws.com
      prefix: stash-qa/demo/deployment/stash-demo
    storageSecretName: s3-secret
status:
  backupCount: 6
  firstBackupTime: 2018-11-14T11:42:40Z
  lastBackupDuration: 3.40542734s
  lastBackupTime: 2018-11-14T11:47:40Z
```

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup snapshots are created successfully by,

```console
$ kubectl get snapshots -n demo -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-2e9cc755   4m53s
deployment.stash-demo-72b5ad8a   3m53s
deployment.stash-demo-e819580d   2m53s
deployment.stash-demo-57386fe3   113s
deployment.stash-demo-c8312c62   53s
```

Here, we can see 5 last successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

If you navigate to `<bucket name>/demo/deployment/stash-demo` directory in your s3 bucket. You will see, a repository has been created there.
<p align="center">
  <img alt="Repository in AWS S3 Backend",  height="350px" src="/docs/images/platforms/eks/s3-backup-repository.png">
</p>

To view the snapshot files, navigate to `snapshots` directory of the repository,

<p align="center">
  <img alt="Snapshot in AWS S3 Bucket" height="350px" src="/docs/images/platforms/eks/s3-backup-snapshots.png">
</p>

> Stash keeps all backup data encrypted. So, snapshot files in the bucket will not contain any meaningful data until they are decrypted.

## Recovery

Now, consider that we have lost our workload as well as data volume. We want to recover the data into a new volume and re-deploy the workload.

At first, let's delete `Restic` crd, `stash-demo` deployment and `stash-sample-data` ConfigMap.

```console
$ kubectl delete deployment -n demo stash-demo
deployment.extensions "stash-demo" deleted

$ kubectl delete restic -n demo s3-restic
restic.stash.appscode.com "s3-restic" deleted

$ kubectl delete configmap -n demo stash-sample-data
configmap "stash-sample-data" deleted
```

In order to perform recovery, we need `Repository` crd `deployment.stah-demo` and backend secret `s3-secret` to exist.

> In case of cluster disaster, you might lose `Repository` crd and backend secret. In this scenario, you have to create the secret again and `Repository` crd manually. Follow the guide to understand `Repository` crd structure from [here](/docs/concepts/crds/repository.md).

**Create PVC:**

We will recover backed up data into a PVC. At first, we need to know available [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) in our cluster.

```
$ kubectl get storageclass
NAME            PROVISIONER             AGE
gp2 (default)   kubernetes.io/aws-ebs   6h
```

Now, let's create a `PersistentVolumeClaim` where our recovered data will be stored.

```console
$ kubectl apply -f ./docs/examples/platforms/eks/pvc.yaml
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
  storageClassName: gp2
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
stash-recovered   Bound    pvc-d86e3909-e80e-11e8-a7b0-029842a88ece   1Gi        RWO            gp2            18s
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-d86e3909-e80e-11e8-a7b0-029842a88ece`.

**Create Recovery:**

Now, we have to create a `Recovery` crd to recover backed up data into this PVC.

```console
$ kubectl apply -f ./docs/examples/platforms/eks/recovery.yaml
recovery.stash.appscode.com/s3-recovery created
```

Below, the YAML for `Recovery` crd we have created above.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: s3-recovery
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

```yaml
$ kubectl get recovery -n demo s3-recovery -o yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"stash.appscode.com/v1alpha1","kind":"Recovery","metadata":{"annotations":{},"name":"s3-recovery","namespace":"demo"},"spec":{"paths":["/source/data"],"recoveredVolumes":[{"mountPath":"/source/data","persistentVolumeClaim":{"claimName":"stash-recovered"}}],"repository":{"name":"deployment.stash-demo","namespace":"demo"}}}
  clusterName: ""
  creationTimestamp: 2018-11-14T13:13:13Z
  generation: 1
  name: s3-recovery
  namespace: demo
  resourceVersion: "42086"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/demo/recoveries/s3-recovery
  uid: 0a843823-e80f-11e8-9c28-06995e414a88
spec:
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    persistentVolumeClaim:
      claimName: stash-recovered
  repository:
    name: deployment.stash-demo
    namespace: demo
status:
  phase: Succeeded

```

Here, `status.phase: Succeeded` indicate that our recovery has been completed successfully. Backup data has been restored in `stash-recovered` PVC. Now, we are ready to use this PVC to re-deploy workload.

**Re-deploy Workload:**

We have successfully restored backup data into `stash-recovered` PVC. Now, we are going to re-deploy our previous deployment `stash-demo`. This time, we are going to mount the `stash-recovered` PVC as `source-data` volume instead of ConfigMap `stash-sample-data`.

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
$  kubectl apply -f ./docs/examples/platforms/eks/recovered-deployment.yaml
deployment.apps/stash-demo created
```

**Verify Recovered Data:**

We have re-deployed `stash-demo` deployment with recovered volume. Now, it is time to verify that the data are present in `/source/data` directory.

Get the pod of new deployment,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-6796866bb8-zkhv5   1/1     Running   0          55s
```

Run following command to view data of `/source/data` directory of this pod,

```console
$ kubectl exec -n demo stash-demo-6796866bb8-zkhv5 -- ls -R /source/data
/source/data:
LICENSE
README.md
lost+found

/source/data/lost+found:
```

So, we can see that the data we had backed up from original deployment are now present in re-deployed deployment.

## Cleanup

To cleanup the resources created by this tutorial, run following commands:

```console
$ kubectl delete recovery -n demo s3-recovery
$ kubectl delete secret -n demo s3-secret
$ kubectl delete deployment -n demo stash-demo
$ kubectl delete pvc -n demo stash-recovered
$ kubectl delete repository -n demo deployment.stash-demo
$ kubectl delete ns demo
```

- To uninstall Stash from your cluster, follow the instructions from [here](/docs/setup/uninstall.md).

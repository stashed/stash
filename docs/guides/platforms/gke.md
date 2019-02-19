---
title: GKE | Stash
description: Using Stash in Google Kubernetes Engine
menu:
  product_stash_0.8.3:
    identifier: platforms-gke
    name: GKE
    parent: platforms
    weight: 30
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Using Stash with Google Kubernetes Engine (GKE)

This tutorial will show you how to use Stash to **backup** and **restore** a Kubernetes deployment in [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/). Here, we are going to back up the `/source/data` folder of a busybox pod into [GCS bucket](/docs/guides/backends.md#google-cloud-storage-gcs). Then, we are going to show how to recover this data into a `gcePersistentDisk` and `PersistentVolumeClaim`. We are going to also re-deploy deployment using this recovered volume.

## Before You Begin

At first, you need to have a Kubernetes cluster in Google Cloud Platform. If you don't already have a cluster, you can create one from [here](https://console.cloud.google.com/kubernetes).

- Install Stash in your cluster following the steps [here](/docs/setup/install.md).

- You should be familiar with the following Stash concepts:
  - [Restic](/docs/concepts/crds/restic.md)
  - [Repository](/docs/concepts/crds/repository.md)
  - [Recovery](/docs/concepts/crds/recovery.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

- You will need a [GCS Bucket](https://console.cloud.google.com/storage) and [GCE persistent disk](https://console.cloud.google.com/compute/disks). GCE persistent disk must be in the same GCE project and zone as the cluster.

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/platforms/gke](/docs/examples/platforms/gke) directory of [appscode/stash](https://github.com/appscode/stash) repository.

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
$ kubectl apply -f ./docs/examples/platforms/gke/deployment.yaml
deployment.apps/stash-demo created
```

Now, wait for deployment's pod to go in `Running` state.

```console
$ kubectl get pods -n demo -l app=stash-demo
NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-b66b9cdfd-8s98d   1/1       Running   0          6m
```

You can check that the `/source/data/` directory of pod is populated with data from the volume source using this command,

```console
$ kubectl exec -n demo stash-demo-b66b9cdfd-8s98d -- ls -R /source/data/
/source/data:
LICENSE
README.md
```

Now, we are ready backup `/source/data` directory into a [GCS bucket](/docs/guides/backends.md#google-cloud-storage-gcs),

**Create Secret:**

At first, we need to create a storage secret that hold the credentials for the backend. To configure this backend, the following secret keys are needed:

|                Key                |                        Description                         |
| --------------------------------- | ---------------------------------------------------------- |
| `RESTIC_PASSWORD`                 | `Required`. Password used to encrypt snapshots by `restic` |
| `GOOGLE_PROJECT_ID`               | `Required`. Google Cloud project ID                        |
| `GOOGLE_SERVICE_ACCOUNT_JSON_KEY` | `Required`. Google Cloud service account json key          |

Create storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-project-id>' > GOOGLE_PROJECT_ID
$ cat downloaded-sa-json.key > GOOGLE_SERVICE_ACCOUNT_JSON_KEY
$ kubectl create secret generic -n demo gcs-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./GOOGLE_PROJECT_ID \
    --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret "gcs-secret" created
```

Verify that the secret has been created successfully,

```console
$ kubectl get secret -n demo gcs-secret -o yaml
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
  namespace: demo
  resourceVersion: "7113"
  selfLink: /api/v1/namespaces/demo/secrets/gcs-secret
  uid: d5e70521-3d87-11e8-a5b9-42010a800002
type: Opaque
```

**Create Restic:**

Now, we can create `Restic` crd. This will create a repository in the GCS bucket specified in `gcs.bucket` field and start taking periodic backup of `/source/data` directory.

```console
$ kubectl apply -f ./docs/examples/platforms/gke/restic.yaml
restic.stash.appscode.com/gcs-restic created
```

Below, the YAML for Restic crd we have created above,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: gcs-restic
  namespace: demo
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

If everything goes well, Stash will inject a sidecar container into the `stash-demo` deployment to take periodic backup. Let's check sidecar has been injected successfully,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-6b8c94cdd7-8jhtn   2/2     Running   1          1h
```

Look at the pod. It now has 2 containers. If you view the resource definition of this pod, you will see there is a container named `stash` which running `backup` command.

**Verify Backup:**

Stash will create a `Repository` crd with name `deployment.stash-demo` for the respective repository in GCS backend. To verify, run the following command,

```console
$ kubectl get repository deployment.stash-demo -n demo
NAME                    BACKUPCOUNT   LASTSUCCESSFULBACKUP   AGE
deployment.stash-demo   1             13s                    1m
```

Here, `BACKUPCOUNT` field indicates number of backup snapshot has taken in this repository.

`Restic` will take backup of the volume periodically with a 1-minute interval. You can verify that backup is taking successfully by,

```console
$ kubectl get snapshots -n demo -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-c1014ca6   10s
```

Here, `deployment.stash-demo-c1014ca6` represents the name of the successful backup [Snapshot](/docs/concepts/crds/snapshot.md) taken by Stash in `deployment.stash-demo` repository.

If you navigate to `<bucket name>/demo/deployment/stash-demo` directory in your GCS bucket. You will see, a repository has been created there.

<p align="center">
  <img alt="Repository in GCS Bucket", src="/docs/images/platforms/gke/gcs-backup-repository.png">
</p>

To view the snapshot files, navigate to `snapshots` directory of the repository,

<p align="center">
  <img alt="Snapshot in GCS Bucket" src="/docs/images/platforms/gke/gcs-backup-snapshots.png">
</p>

> Stash keeps all backup data encrypted. So, snapshot files in the bucket will not contain any meaningful data until they are decrypted.

## Recovery

Now, consider that we have lost our workload as well as data volume. We want to recover the data into a new volume and re-deploy the workload. In this section, we are going to see how to recover data into a  [gcePersistentDisk](https://kubernetes.io/docs/concepts/storage/volumes/#gcepersistentdisk) and [persistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim).

At first, let's delete `Restic` crd, `stash-demo` deployment and `stash-sample-data` ConfigMap.

```console
$ kubectl delete deployment -n demo stash-demo
deployment.extensions "stash-demo" deleted

$ kubectl delete restic -n demo gcs-restic
restic.stash.appscode.com "gcs-restic" deleted

$ kubectl delete configmap -n demo stash-sample-data
configmap "stash-sample-data" deleted
```

In order to perform recovery, we need `Repository` crd `deployment.stah-demo` and backend secret `gcs-secret` to exist.

>In case of cluster disaster, you might lose `Repository` crd and backend secret. In this scenario, you have to create the secret again and `Repository` crd manually. Follow the guide to understand `Repository` crd structure from [here](/docs/concepts/crds/repository.md).

### Recover to GCE Persistent Disk

Now, we are going to recover the backed up data into GCE Persistent Disk. At first, create a GCE disk named `stash-recovered` from [Google cloud console](https://console.cloud.google.com/compute/disks). Then create `Recovery` crd,

```console
$ kubectl apply -f ./docs/examples/platforms/gke/recovery-gcePD.yaml
recovery.stash.appscode.com/gcs-recovery created
```

Below, the YAML for `Recovery` crd we have created above.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: gcs-recovery
  namespace: demo
spec:
  repository:
    name: deployment.stash-demo
    namespace: demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    gcePersistentDisk:
        pdName: stash-recovered
        fsType: ext4
```

Wait until `Recovery` job completes its task. To verify that recovery has completed successfully run,

```console
$ kubectl get recovery -n demo gcs-recovery
NAME             REPOSITORYNAMESPACE   REPOSITORYNAME          SNAPSHOT   PHASE       AGE
gcs-recovery     demo                  deployment.stash-demo              Succeeded   3m
```

Here, `PHASE` `Succeeded` indicate that our recovery has been completed successfully. Backup data has been restored in `stash-recovered` Persistent Disk. Now, we are ready to use this Persistent Disk to re-deploy workload.

If you are using Kubernetes version older than v1.11.0 then run following command and check `status.phase` field to see whether the recovery succeeded or failed.

```console
$ kubectl get recovery -n demo gcs-recovery -o yaml
```

**Re-deploy Workload:**

We have successfully restored backup data into `stash-recovered` gcePersistentDisk. Now, we are going to re-deploy our previous deployment `stash-demo`. This time, we are going to mount the `stash-recovered` Persistent Disk as `source-data` volume instead of ConfigMap `stash-sample-data`.

Below, the YAML for `stash-demo` deployment with `stash-recovered` persistent disk as `source-data` volume.

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
        gcePersistentDisk:
          pdName: stash-recovered
          fsType: ext4
```

Let's create the deployment,

```console
$  kubectl apply -f ./docs/examples/platforms/gke/restored-deployment-gcePD.yaml
deployment.apps/stash-demo created
```

**Verify Recovered Data:**

We have re-deployed `stash-demo` deployment with recovered volume. Now, it is time to verify that the data are present in `/source/data` directory.

Get the pod of new deployment,

```console
$ kubectl get pods -n demo -l app=stash-demo
NAME                         READY     STATUS    RESTARTS   AGE
stash-demo-857995799-gpml9   1/1       Running   0          34s
```

Run following command to view data of `/source/data` directory of this pod,

```console
$ kubectl exec -n demo stash-demo-857995799-gpml9 -- ls -R /source/data
/source/data:
LICENSE
README.md
lost+found

/source/data/lost+found:
```

So, we can see that the data we had backed up from original deployment are now present in re-deployed deployment.

### Recover to `PersistentVolumeClaim`

Here, we are going to show how to recover the backed up data into a PVC. If you have re-deployed `stash-demo` deployment by following previous tutorial on `gcePersistentDisk`, delete the deployment first,

```console
$ kubectl delete deployment -n demo stash-demo
deployment.apps/stash-demo deleted
```

Now, create a `PersistentVolumeClaim` where our recovered data will be stored.

```console
$ kubectl apply -f ./docs/examples/platforms/gke/pvc.yaml
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
      storage: 2Gi
```

Check that if cluster has provisioned the requested claim,

```console
$ kubectl get pvc -n demo -l app=stash-demo
NAME              STATUS    VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
stash-recovered   Bound     pvc-57bec6e5-3e11-11e8-951b-42010a80002e   2Gi        RWO            standard       1m
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-57bec6e5-3e11-11e8-951b-42010a80002e`.

**Create Recovery:**

Now, we have to create a `Recovery` crd to recover backed up data into this PVC.

```console
$ kubectl apply -f ./docs/examples/platforms/gke/recovery-pvc.yaml
recovery.stash.appscode.com/gcs-recovery created
```

Below, the YAML for `Recovery` crd we have created above.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: gcs-recovery
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
$ kubectl get recovery -n demo gcs-recovery
NAME             REPOSITORYNAMESPACE   REPOSITORYNAME          SNAPSHOT   PHASE       AGE
gcs-recovery     demo                  deployment.stash-demo              Succeeded   3m
```

Here, `PHASE` `Succeeded` indicate that our recovery has been completed successfully. Backup data has been restored in `stash-recovered` PVC. Now, we are ready to use this PVC to re-deploy workload.

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
$ kubectl apply -f ./docs/examples/platforms/gke/restored-deployment-pvc.yaml
deployment.apps/stash-demo created
```

**Verify Recovered Data:**

We have re-deployed `stash-demo` deployment with recovered volume. Now, it is time to verify that the data are present in `/source/data` directory.

Get the pod of new deployment,

```console
$ kubectl get pods -n demo -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-559845c5db-8cd4w   1/1       Running   0          33s
```

Run following command to view data of `/source/data` directory of this pod,

```console
$ kubectl exec -n demo stash-demo-559845c5db-8cd4w -- ls -R /source/data
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
$ kubectl delete recovery -n demo gcs-recovery
$ kubectl delete secret -n demo gcs-secret
$ kubectl delete deployment -n demo stash-demo
$ kubectl delete pvc -n demo stash-recovered
$ kubectl delete repository -n demo deployment.stash-demo
$ kubectl delete ns demo
```

- To uninstall Stash from your cluster, follow the instructions from [here](/docs/setup/uninstall.md).

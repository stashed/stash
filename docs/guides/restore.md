---
title: Restore Volumes | Stash
description: Restore Volumes using Stash
menu:
  product_stash_0.8.0:
    identifier: restore-stash
    name: Restore Volumes
    parent: guides
    weight: 20
product_name: stash
menu_name: product_stash_0.8.0
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Restore from Backup

This tutorial will show you how to restore backed up volume using Stash. Here, we are going to recover backed up data into a PVC. Then, we are going to re-deploy the workload using the recovered volume.

## Before You Begin

To proceed with this tutorial, you have to meet following requirements:

- At first, you need to have some backup taken by Stash. If you already don't have any backup repository, create one by following this [backup tutorial](/docs/guides/backup.md).

- You need to have the storage `Secret` that was used to take backup. If you don't have the `Secret`, create one with valid credentials.

- You need to have `Repository` crd that was created for the respective backup. If you have lost the `Repository` crd, you have to create it manually with respective backend information. Follow, [this guide](/docs/concepts/crds/repository.md) to understand structure of `Repository` crd.

- You should be familiar with the following Stash concepts:
  - [Repository](/docs/concepts/crds/repository.md)
  - [Recovery](/docs/concepts/crds/recovery.md)
  - [Snapshot](/docs/concepts/crds/snapshot.md)

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial. Create the namespace if you haven't created yet.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/recovery](/docs/examples/recovery) directory of [appscode/stash](https://github.com/appscode/stash) repository.

## Overview

The following diagram shows how Stash recovers backed up data from a backend. Open the image in a new tab to see the enlarged image.

<p align="center">
  <img alt="Stash Backup Flow" src="/docs/images/stash-recovery.svg">
</p>

The volume recovery backup process consists of the following steps:

1. A user creates a `Recovery` crd that specifies the target `Repository` from where he/she want to recover. It also specifies one or more volumes (`recoveredVolumes`) where the recovered data will be stored.
2. Stash operator watches for new `Recovery` crds. If it sees one, it checks if the referred `Repository` crd exists or not.
3. Then, Stash operator creates a `Job` to recover the backed up data.
4. The recovery `Job` reads the backend information from `Repository` crd and the backend credentials from the storage `Secret`.
5. Then, the recovery `Job` recovers data from the backend and stores it in the target volume.
6. Finally, the user mounts this recovered volume into the original workload and re-deploys it.

## Recovery

Now, we are going to recover backed up data from `deployment.stash-demo` Repository that was created while taking backup into a PVC named `stash-recovered`.

At first, let's delete `Restic` crd so that it does not lock the repository while are recovering from it. Also, delete `stash-demo` deployment and `stash-sample-data` ConfigMap if you followed our backup guide.

```console
$ kubectl delete deployment -n demo stash-demo
deployment.extensions "stash-demo" deleted

$ kubectl delete restic -n demo local-restic
restic.stash.appscode.com "local-restic" deleted

$ kubectl delete configmap -n demo stash-sample-data
configmap "stash-sample-data" deleted
```

>Note: In order to perform recovery, we need `Repository` crd (in our case `deployment.stash-demo`) and backend secret (in our case `local-secret`).

**Create PVC:**

We will recover backed up data into a PVC. At first, we need to know available [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) in our cluster.

```console
$ kubectl get storageclass
NAME                 PROVISIONER                AGE
standard (default)   k8s.io/minikube-hostpath   8h
```

Now, let's create a `PersistentVolumeClaim` where our recovered data will be stored.

```console
$ kubectl apply -f ./docs/examples/recovery/pvc.yaml
persistentvolumeclaim/stash-recovered created
```

Here is the definition of the `PersistentVolumeClaim` we have created above,

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

Check whether cluster has provisioned the requested claim.

```console
$ kubectl get pvc -n demo -l app=stash-demo
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
stash-recovered   Bound    pvc-e6ffface-fa01-11e8-8905-0800277ca39d   50Mi       RWO            standard       13s
```

Look at the `STATUS` filed. `stash-recovered` PVC is bounded to volume `pvc-e6ffface-fa01-11e8-8905-0800277ca39d`.

**Create Recovery CRD:**

Now, we have to create a `Recovery` crd to recover backed up data into this PVC.

The resource definition of the `Recovery` crd we are going to create is below:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: local-recovery
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

Here,

- `spec.repository.name` specifies the name of the `Repository` crd that represents respective **restic** repository.
- `spec.repository.namespace` specifies the namespace of `Repository` crd.
- `spec.paths` specifies the file-group paths that were backed up using `Restic`.
- `spec.recoveredVolumes` indicates an array of volumes where snapshots will be recovered. Here, `mountPath` specifies where the volume will be mounted. Note that, `Recovery` recovers data in the same paths from where the backup was taken (specified in `spec.paths`). So, volumes must be mounted on those paths or their parent paths.

Let's create the Recovery crd we have shown above,

```console
$ kubectl apply -f ./docs/examples/recovery/recovery.yaml
recovery.stash.appscode.com/local-recovery created
```

Wait until `Recovery` job completes its task. To verify that recovery has completed successfully run,

```console
$ kubectl get recovery -n demo local-recovery
NAME             REPOSITORY-NAMESPACE  REPOSITORY-NAME         SNAPSHOT   PHASE       AGE
local-recovery   demo                  deployment.stash-demo              Succeeded   54s
```

Here, `PHASE` `Succeeded` indicates that our recovery has been completed successfully. Backup data has been restored in `stash-recovered` PVC. Now, we are ready to use this PVC to re-deploy the workload.

If you are using Kubernetes version older than v1.11.0 then run following command and check `status.phase` field to see whether the recovery succeeded or failed.

```console
$ kubectl get recovery -n demo local-recovery -o yaml
```

**Re-deploy Workload:**

We have successfully restored backed up data into `stash-recovered` PVC. Now, we are going to re-deploy our previous deployment `stash-demo`. This time, we are going to mount the `stash-recovered` PVC as `source-data` volume instead of ConfigMap `stash-sample-data`.

Below is the YAML for `stash-demo` deployment with `stash-recovered` PVC as `source-data` volume.

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
$ kubectl apply -f ./docs/examples/recovery/recovered-deployment.yaml
deployment.apps/stash-demo created
```

**Verify Recovered Data:**

We have re-deployed `stash-demo` deployment with recovered volume. Now, it is time to verify that the recovered data are present in `/source/data` directory.

Get the pod of new deployment,

```console
$ kubectl get pod -n demo -l app=stash-demo
NAME                          READY   STATUS    RESTARTS   AGE
stash-demo-69694789df-kvcp5   1/1     Running   0          20s
```

Run following command to view data of `/source/data` directory of this pod,

```console
$ kubectl exec -n demo stash-demo-69694789df-kvcp5 -- ls -R /source/data
/source/data:
LICENSE
README.md
```

So, we can see that the data we had backed up from original deployment are now present in re-deployed deployment.

## Recover a specific snapshot

With the help of [Snapshot](/docs/concepts/crds/snapshot.md) object, Stash allows users to recover from a particular snapshot. Here is an example of how to recover from a specific snapshot.

First, list the available snapshots,

```console
$ kubectl get snapshots -n demo -l repository=deployment.stash-demo
NAME                             AGE
deployment.stash-demo-bd8db133   4m50s
deployment.stash-demo-b6e67dee   3m50s
deployment.stash-demo-10790cf0   2m50s
deployment.stash-demo-1ace430f   110s
deployment.stash-demo-baff6c47   50s
```

>Note: If you are using [Local](/docs/guides/backends/local.md) backend for storing backup snapshots, your workload must be running to be able to list snapshots.

Below is the YAML for `Recovery` crd referring to a specific snapshot.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: local-recovery-specific-snapshot
  namespace: demo
spec:
  repository:
    name: deployment.stash-demo
    namespace: demo
  snapshot: deployment.stash-demo-baff6c47
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    persistentVolumeClaim:
      claimName: stash-recovered
```

Now, create a `Recovery` crd shown above,

```console
$ kubectl apply -f ./docs/examples/recovery/recovery-specific-snapshot.yaml
recovery.stash.appscode.com/local-recovery-specific-snapshot created
```

## Cleanup

To cleanup the resources created by this tutorial, run following commands:

```console
$ kubectl delete recovery -n demo local-recovery
$ kubectl delete recovery -n demo local-recovery-specific-snapshot
$ kubectl delete secret -n demo local-secret
$ kubectl delete deployment -n demo stash-demo
$ kubectl delete pvc -n demo stash-recovered
$ kubectl delete repository -n demo deployment.stash-demo

$ kubectl delete ns demo
```

If you would like to uninstall Stash operator, please follow the steps [here](/docs/setup/uninstall.md).

## Next Steps

- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends/overview.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring/overview.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

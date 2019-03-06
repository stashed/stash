---
title: Backup and Restore PVC | Stash
description: Backup and Restore PVC using Stash
menu:
  product_stash_0.8.3:
    identifier: backup_restore_pvc
    name: Backup and Restore PVC
    parent: guides
    weight: 15
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Backup and Restore PVC

In this tutorial we will backup some directories of a persistent volume and restore it in another persistent volume.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the `kubectl` command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube). Then install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial.

```console
$ kubectl create ns demo
namespace/demo created
```

>Note: YAML files used in this tutorial are stored in [/docs/examples/pvc](/docs/examples/pvc) directory of [appscode/stash](https://github.com/appscode/stash) repository.

## Create Functions and Tasks

```bash
$ kubectl apply -f /docs/examples/pvc/functions.yaml; kubectl apply -f /docs/examples/pvc/tasks.yaml
function.stash.appscode.com/pvc-backup created
task.stash.appscode.com/pvc-backup-task created
function.stash.appscode.com/pvc-restore created
task.stash.appscode.com/pvc-restore-task created
```

## Create Repository

```bash
$ kubectl create secret generic gcs-secret -n demo --from-file=./RESTIC_PASSWORD --from-file=./GOOGLE_PROJECT_ID --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret/gcs-secret created
```

```bash
$ kubectl apply -f /docs/examples/pvc/repo.yaml
repository.stash.appscode.com/hello-repo created
```

## Create PVC

Create Persistent Volume and two separate PVC for source and destination.

```bash
$ kubectl apply -f /docs/examples/pvc/pvc.yaml
persistentvolume/test-pv-volume created
persistentvolumeclaim/test-pvc-source created
persistentvolumeclaim/test-pvc-dest created
```

## Write to Source PVC

```bash
$ kubectl apply -f /docs/examples/pvc/write.yaml
pod/test-write-soruce created
```

## Backup Source PVC

```bash
$ kubectl apply -f /docs/examples/pvc/backup.yaml
backupconfiguration.stash.appscode.com/pvc-backup-config created
backupsession.stash.appscode.com/pvc-backup-01 created
```

## Restore to Destination PVC

```bash
$ kubectl apply -f /docs/examples/pvc/restore.yaml
restoresession.stash.appscode.com/pvc-restore-01 created
```

## Check Destination PVC

```bash
$ kubectl apply -f /docs/examples/pvc/check.yaml
pod/test-check-dest created
```

```bash
$ kc logs -n demo  -f test-check-dest
+ ls /etc/target
dir-01
dir-02
```

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete -f /docs/examples/pvc
$ kubectl delete namespace demo
```

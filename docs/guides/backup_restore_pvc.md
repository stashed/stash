
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

>Note: YAML files used in this tutorial are stored in [./docs/examples/pvc](./docs/examples/pvc) directory of [appscode/stash](https://github.com/stashed/stash) repository.

## Create Service Account

```bash
$ kubectl apply -f ./docs/examples/pvc/rbac.yaml
```

## Create Functions and Tasks

```bash
$ kubectl apply -f ./docs/examples/pvc/functions.yaml
$ kubectl apply -f ./docs/examples/pvc/tasks.yaml
```

## Create Repository

```bash
$ kubectl create secret generic gcs-secret -n demo --from-file=./RESTIC_PASSWORD --from-file=./GOOGLE_PROJECT_ID --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret/gcs-secret created
```

```bash
$ kubectl apply -f ./docs/examples/pvc/repo.yaml
repository.stash.appscode.com/hello-repo created
```

## Create PVC

Create Persistent Volume and two separate PVC for source and destination.

```bash
$ kubectl apply -f ./docs/examples/pvc/pvc.yaml
persistentvolume/test-pv-volume created
persistentvolumeclaim/test-pvc-source created
persistentvolumeclaim/test-pvc-dest created
```

## Write to Source PVC

```bash
$ kubectl apply -f ./docs/examples/pvc/write.yaml
pod/test-write-soruce created
```

## Backup Source PVC

```bash
$ kubectl apply -f ./docs/examples/pvc/backup.yaml
backupconfiguration.stash.appscode.com/pvc-backup-config created
backupsession.stash.appscode.com/pvc-backup-01 created
```

## Check Backup Session Status

```yaml
  status:
    phase: Succeeded
    stats:
    - directory: /etc/target/dir-01
      fileStats:
        modifiedFiles: 0
        newFiles: 0
        totalFiles: 0
        unmodifiedFiles: 0
      processingTime: 0m7s
      size: 0 B
      snapshot: f5268c39
      uploaded: 336 B
    - directory: /etc/target/dir-02
      fileStats:
        modifiedFiles: 0
        newFiles: 0
        totalFiles: 0
        unmodifiedFiles: 0
      processingTime: 0m7s
      size: 0 B
      snapshot: e6277c72
      uploaded: 336 B
```

## Check Repository Status

```yaml
  status:
    integrity: true
    size: 5.913 KiB
    snapshotCount: 10
    snapshotsRemovedOnLastCleanup: 2
```

## Restore to Destination PVC

```bash
$ kubectl apply -f ./docs/examples/pvc/restore.yaml
restoresession.stash.appscode.com/pvc-restore-01 created
```

## Check Restore Session Status

```yaml
  status:
    duration: 31.229339267s
    phase: Succeeded
```

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete -f ./docs/examples/pvc
$ kubectl delete namespace demo
```

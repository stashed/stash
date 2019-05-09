---
title: Backend Overview | Stash
description: An overview of backends used by Stash to store snapshot data.
menu:
  product_stash_0.8.3:
    identifier: backend-overview
    name: What is Backend?
    parent: v1alpha1-backend
    weight: 10
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Stash Backends

Backend is where Stash stores backup snapshots. It can be a cloud storage like GCS bucket, AWS S3, Azure Blob Storage etc. or a Kubernetes persistent volume like [HostPath](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath), [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim), [NFS](https://kubernetes.io/docs/concepts/storage/volumes/#nfs) etc. Below diagram show how Stash sidecar container access and store backup data into a backend storage.

<p align="center">
  <img alt="Stash Backup Overview" height="350px", src="/docs/images/v1alpha1/backup-overview.png">
</p>

Stash sidecar container receive backend information from `spec.backend` field of [Restic](/docs/concepts/crds/restic.md) crd. It obtains necessary credentials to access the backend from the secret specified in `spec.backend.storageSecretName` field of Restic crd. Then on first backup schedule, Stash initialize a repository in the backend.

Below, a screenshot that show a repository created at AWS S3 bucket named `stash-qa` for a Deployment named `stash-demo`.

<p align="center">
  <img alt="Repository in AWS S3 Backend", src="/docs/images/v1alpha1/platforms/eks/s3-backup-repository.png">
</p>

You will see all snapshots taken by Stash at `/snapshot` directory of this repository.

> Note: Stash keeps all backup data encrypted. So, snapshot files in the bucket will not contain any meaningful data until they are decrypted.

Stash creates a [Repository](/docs/concepts/crds/repository.md) crd that represents original repository in backend in Kubernetes native way. It holds information like number of backup snapshot taken, time when last backup was taken etc.

In order to use a backend, you have to configure `Restic` crd and create a `Secret` with necessary credentials.

## Next Steps

- Learn how to configure `local` backend from [here](/docs/guides/v1alpha1/backends/local.md).
- Learn how to configure `AWS S3` backend from [here](/docs/guides/v1alpha1/backends/s3.md).
- Learn how to configure `Google Cloud Storage (GCS)` backend from [here](/docs/guides/v1alpha1/backends/gcs.md).
- Learn how to configure `Microsoft Azure Storage` backend from [here](/docs/guides/v1alpha1/backends/azure.md).
- Learn how to configure `OpenStack Swift` backend from [here](/docs/guides/v1alpha1/backends/swift.md).
- Learn how to configure `Backblaze B2` backend from [here](/docs/guides/v1alpha1/backends/b2.md).

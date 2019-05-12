---
title: GCS Backend | Stash
description: Configure Stash to use Google Cloud Storage (GCS) as Backend.
menu:
  product_stash_0.8.3:
    identifier: v1alpha1-backend-gcs
    name: Google Cloud Storage
    parent: v1alpha1-backend
    weight: 50
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

# Google Cloud Storage (GCS)

Stash supports Google Cloud Storage(GCS) as backend. This tutorial will show you how to configure **Restic** and storage **Secret** for GCS backend.

#### Create Storage Secret

To configure storage secret for this backend, following secret keys are needed:

| Key                               | Description                                                |
|-----------------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`                 | `Required`. Password used to encrypt snapshots by `restic` |
| `GOOGLE_PROJECT_ID`               | `Required`. Google Cloud project ID                        |
| `GOOGLE_SERVICE_ACCOUNT_JSON_KEY` | `Required`. Google Cloud service account json key          |

Create storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-project-id>' > GOOGLE_PROJECT_ID
$ mv downloaded-sa-json.key > GOOGLE_SERVICE_ACCOUNT_JSON_KEY
$ kubectl create secret generic gcs-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./GOOGLE_PROJECT_ID \
    --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret "gcs-secret" created
```

Verify that the secret has been created with respective keys,

```yaml
$ kubectl get secret gcs-secret -o yaml

apiVersion: v1
data:
  GOOGLE_PROJECT_ID: PHlvdXItcHJvamVjdC1pZD4=
  GOOGLE_SERVICE_ACCOUNT_JSON_KEY: ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3V...9tIgp9Cg==
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-06-28T13:06:51Z
  name: gcs-secret
  namespace: default
  resourceVersion: "5461"
  selfLink: /api/v1/namespaces/default/secrets/gcs-secret
  uid: a6983b00-5c02-11e7-bb52-08002711f4aa
type: Opaque
```

#### Configure Restic

Now, you have to configure Restic crd to use GCS bucket. You have to provide previously created storage secret in `spec.backend.storageSecretName` field.

Following parameters are available for `gcs` backend.

| Parameter      | Description                                                                     |
|----------------|---------------------------------------------------------------------------------|
| `gcs.bucket`   | `Required`. Name of Bucket. If the bucket does not exist yet, it will be created in the default location (US). It is not possible at the moment to have restic create a new bucket in a different location, so you need to create it using a different program.        |
| `gcs.prefix`   | `Optional`. Path prefix into bucket where repository will be created.           |

Below, the YAML for Restic crd configured to use GCS bucket.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: gcs-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: gcs-restic
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    gcs:
      bucket: stash-qa
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

Now, create the Restic we have configured above for `gcs` backend,

```console
$ kubectl apply -f ./docs/examples/backends/gcs/gcs-restic.yaml
restic "gcs-restic" created
```

## Next Steps

- Learn how to use Stash in Google Kubernetes Engine (GKE) from [here](/docs/guides/v1alpha1/platforms/gke.md).
- Learn how to use Stash to backup a Kubernetes deployment from [here](/docs/guides/v1alpha1/backup.md).
- Learn how to recover from backed up snapshot from [here](/docs/guides/v1alpha1/restore.md).

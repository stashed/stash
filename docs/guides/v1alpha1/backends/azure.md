---
title: Azure Backend | Stash
description: Configure Stash to use Microsoft Azure Storage as Backend.
menu:
  product_stash_0.8.3:
    identifier: v1alpha1-backend-azure
    name: Azure Blob Storage
    parent: v1alpha1-backend
    weight: 40
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

# Microsoft Azure Storage

Stash supports Microsoft Azure Storage as backend. This tutorial will show you how to configure **Restic** and storage **Secret** for Azure Storage.

#### Create Storage Secret

To configure storage secret for this backend, following secret keys are needed:

| Key                     | Description                                                |
|-------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AZURE_ACCOUNT_NAME`    | `Required`. Azure Storage account name                     |
| `AZURE_ACCOUNT_KEY`     | `Required`. Azure Storage account key                      |

Create storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-azure-storage-account-name>' > AZURE_ACCOUNT_NAME
$ echo -n '<your-azure-storage-account-key>' > AZURE_ACCOUNT_KEY
$ kubectl create secret generic azure-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./AZURE_ACCOUNT_NAME \
    --from-file=./AZURE_ACCOUNT_KEY
secret "azure-secret" created
```

Verify that the secret has been created with respective keys,

```yaml
$ kubectl get secret azure-secret -o yaml

apiVersion: v1
data:
  AZURE_ACCOUNT_KEY: PHlvdXItYXp1cmUtc3RvcmFnZS1hY2NvdW50LWtleT4=
  AZURE_ACCOUNT_NAME: PHlvdXItYXp1cmUtc3RvcmFnZS1hY2NvdW50LW5hbWU+
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-06-28T13:27:16Z
  name: azure-secret
  namespace: default
  resourceVersion: "6809"
  selfLink: /api/v1/namespaces/default/secrets/azure-secret
  uid: 80f658d1-5c05-11e7-bb52-08002711f4aa
type: Opaque
```

#### Configure Restic

Now, you have to configure Restic crd to use Azure Storage. You have to provide previously created storage secret in `spec.backend.storageSecretName` field.

Following parameters are available for `Azure` backend.

|     Parameter     |                              Description                              |
| ----------------- | --------------------------------------------------------------------- |
| `azure.container` | `Required`. Name of Storage container                                 |
| `azure.prefix`    | `Optional`. Path prefix into bucket where repository will be created. |

Below, the YAML for Restic crd configured to use Azure Storage.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: azure-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: azure-restic
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    azure:
      container: stashqa
      prefix: demo
    storageSecretName: azure-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Now, create the Restic we have configured above for `azure` backend,

```console
$ kubectl apply -f ./docs/examples/backends/azure/azure-restic.yaml
restic "azure-restic" created
```

## Next Steps

- Learn how to use Stash in Azure Kubernetes Service (AKS) from [here](/docs/guides/v1alpha1/platforms/aks.md).
- Learn how to use Stash to backup a Kubernetes deployment from [here](/docs/guides/v1alpha1/backup.md).
- Learn how to recover from backed up snapshot from [here](/docs/guides/v1alpha1/restore.md).

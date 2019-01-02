---
title: Local Backend | Stash
description: Configure Stash to Use Local Backend.
menu:
  product_stash_0.8.2:
    identifier: backend-local
    name: Persistent Volumes
    parent: backend
    weight: 20
product_name: stash
menu_name: product_stash_0.8.2
section_menu_id: guides
---

# Local

`Local` backend refers to a local path inside `stash` sidecar container. Any Kubernetes supported [persistent volume](https://kubernetes.io/docs/concepts/storage/volumes/) can be used here. Some examples are: emptyDir (for testing), NFS, Ceph, GlusterFS, etc. This tutorial will show you how to configure **Restic** and storage **Secret** for Local backend.

#### Create Storage Secret

To configure storage secret for this backend, following secret keys are needed:

| Key               | Description                                                |
|-------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD` | `Required`. Password used to encrypt snapshots by `restic` |

Create storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ kubectl create secret generic local-secret --from-file=./RESTIC_PASSWORD
secret "local-secret" created
```
Verify that the secret has been created with respective keys,

```yaml
$ kubectl get secret local-secret -o yaml

apiVersion: v1
data:
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-06-28T12:06:19Z
  name: stash-local
  namespace: default
  resourceVersion: "1440"
  selfLink: /api/v1/namespaces/default/secrets/stash-local
  uid: 31a47380-5bfa-11e7-bb52-08002711f4aa
type: Opaque
```

#### Configure Restic

Now, you have to configure Restic crd to use Local backend. You have to provide previously created storage secret in `spec.backend.storageSecretName` field.

Following parameters are available for `Local` backend.

| Parameter            | Description                                                                                   |
|----------------------|-----------------------------------------------------------------------------------------------|
| `local.mountPath`    | `Required`. Path where this volume will be mounted in the sidecar container. Example: `/repo` |
| `local.subPath`      | `Optional`. Sub-path inside the referenced volume instead of its root.                        |
| `local.VolumeSource` | `Required`. Any Kubernetes volume. Can be specified inlined. Example: `hostPath`              |

**Sample Restic for `hostPath` as Backend :**

Below, the YAML for Restic crd configured to use `hostPath` as Local backend.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: local-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: local-restic
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /repo
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: local-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Now, create the `Restic` we have configured above for `local` backend,

```console
$ kubectl apply -f ./docs/examples/backends/local/local-restic-hostPath.yaml
restic "local-restic" created
```

**Sample Restic for `NFS` Server as Backend :**

Below, the YAML for Restic crd configured to use a `NFS` server as Local backend.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: local-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    local:
      mountPath: /safe/data
      nfs:
        server: "nfs-service.storage.svc.cluster.local" # use you own NFS server address
        path: "/" # this path is relative to "/exports" path of NFS server
    storageSecretName: local-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Now, create the `Restic` we have configured above for `local` backend,

```console
$ kubectl apply -f ./docs/examples/backends/local/local-restic-nfs.yaml
restic "local-restic" created
```

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment from [here](/docs/guides/backup.md).
- To learn how to recover from backed up snapshot, visit [here](/docs/guides/restore.md).

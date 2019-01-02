---
title: Swift Backend | Stash
description: Configure Stash to use OpenStack Swift as Backend.
menu:
  product_stash_0.8.2:
    identifier: backend-swift
    name: OpenStack Swift
    parent: backend
    weight: 60
product_name: stash
menu_name: product_stash_0.8.2
section_menu_id: guides
---

# OpenStack Swift

Stash supports [OpenStack Swift as backend](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#openstack-swift). This tutorial will show you how to configure **Restic** and storage **Secret** for Swift backend.

#### Create Storage Secret

To configure storage secret this backend, following secret keys are needed:

| Key                      | Description                                                |
|--------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`        | `Required`. Password used to encrypt snapshots by `restic` |
| `ST_AUTH`                | For keystone v1 authentication                             |
| `ST_USER`                | For keystone v1 authentication                             |
| `ST_KEY`                 | For keystone v1 authentication                             |
| `OS_AUTH_URL`            | For keystone v2 authentication                             |
| `OS_REGION_NAME`         | For keystone v2 authentication                             |
| `OS_USERNAME`            | For keystone v2 authentication                             |
| `OS_PASSWORD`            | For keystone v2 authentication                             |
| `OS_TENANT_ID`           | For keystone v2 authentication                             |
| `OS_TENANT_NAME`         | For keystone v2 authentication                             |
| `OS_AUTH_URL`            | For keystone v3 authentication                             |
| `OS_REGION_NAME`         | For keystone v3 authentication                             |
| `OS_USERNAME`            | For keystone v3 authentication                             |
| `OS_PASSWORD`            | For keystone v3 authentication                             |
| `OS_USER_DOMAIN_NAME`    | For keystone v3 authentication                             |
| `OS_PROJECT_NAME`        | For keystone v3 authentication                             |
| `OS_PROJECT_DOMAIN_NAME` | For keystone v3 authentication                             |
| `OS_STORAGE_URL`         | For authentication based on tokens                         |
| `OS_AUTH_TOKEN`          | For authentication based on tokens                         |

Create storage secret as below,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-auth-url>' > OS_AUTH_URL
$ echo -n '<your-tenant-id>' > OS_TENANT_ID
$ echo -n '<your-tenant-name>' > OS_TENANT_NAME
$ echo -n '<your-username>' > OS_USERNAME
$ echo -n '<your-password>' > OS_PASSWORD
$ echo -n '<your-region>' > OS_REGION_NAME
$ kubectl create secret generic swift-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./OS_AUTH_URL \
    --from-file=./OS_TENANT_ID \
    --from-file=./OS_TENANT_NAME \
    --from-file=./OS_USERNAME \
    --from-file=./OS_PASSWORD \
    --from-file=./OS_REGION_NAME
secret "swift-secret" created
```

Verify that the secret has been created with respective keys,

```yaml
$ kubectl get secret swift-secret -o yaml

apiVersion: v1
data:
  OS_AUTH_URL: PHlvdXItYXV0aC11cmw+
  OS_PASSWORD: PHlvdXItcGFzc3dvcmQ+
  OS_REGION_NAME: PHlvdXItcmVnaW9uPg==
  OS_TENANT_ID: PHlvdXItdGVuYW50LWlkPg==
  OS_TENANT_NAME: PHlvdXItdGVuYW50LW5hbWU+
  OS_USERNAME: PHlvdXItdXNlcm5hbWU+
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-07-03T19:17:39Z
  name: swift-secret
  namespace: default
  resourceVersion: "36381"
  selfLink: /api/v1/namespaces/default/secrets/swift-secret
  uid: 47b4bcab-6024-11e7-879a-080027726d6b
type: Opaque
```

#### Configure Restic

Now, you have to configure Restic crd to use OpenStack swift backend. You have to provide previously created storage secret in `spec.backend.storageSecretName` field.

Following parameters are available for `Swift` backend.

| Parameter         | Description                                                                 |
|-------------------|-----------------------------------------------------------------------------|
| `swift.container` | `Required`. Name of Storage container                                       |
| `swift.prefix`    | `Optional`. Path prefix into bucket where repository will be created.       |

Below, the YAML for Restic crd configured to use Swift backend.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: swift-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: swift-restic
  fileGroups:
  - path: /source/data
    retentionPolicyName: 'keep-last-5'
  backend:
    swift:
      container: stashqa
      prefix: demo
    storageSecretName: swift-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Now, create the Restic we have configured above for `swift` backend,

```console
$ kubectl apply -f ./docs/examples/backends/swift/swift-restic.yaml
restic "swift-restic" created
```

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment from [here](/docs/guides/backup.md).
- Learn how to recover from backed up snapshot from [here](/docs/guides/restore.md).
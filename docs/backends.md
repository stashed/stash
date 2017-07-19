> New to Stash? Please start [here](/docs/tutorial.md).

# Stash Backends
Backend is where `restic` stores snapshots. For any backend, a Kubernetes Secret in the same namespace is needed to provide restic repository credentials. This Secret can be configured by setting `spec.backend.storageSecretName` field. This document lists the various supported backends for Stash and how to configure those.

### Local
`Local` backend refers to a local path inside `stash` sidecar container. Any Kubernetes supported [persistent volume](https://kubernetes.io/docs/concepts/storage/volumes/) can be used here. Some examples are: `emptyDir` for testing, NFS, Ceph, GlusterFS, etc. To configure this backend, following secret keys are needed:

| Key               | Description                                                |
|-------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD` | `Required`. Password used to encrypt snapshots by `restic` |

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ kubectl create secret generic local-secret --from-file=./RESTIC_PASSWORD
secret "local-secret" created
```

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

Now, you can create a Restic tpr using this secret. Following parameters are available for `Local` backend.

| Parameter      | Description                                                                                 |
|----------------|---------------------------------------------------------------------------------------------|
| `local.path`   | `Required`. Path where this volume will be mounted in the sidecar container. Example: /repo |
| `local.volume` | `Required`. Any Kubernetes volume                                                           |

```console
$ kubectl create -f ./docs/examples/backends/local/local-restic.yaml
restic "local-restic" created
```

```yaml
$ kubectl get restic local-restic -o yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T12:14:48Z
  name: local-restic
  namespace: default
  resourceVersion: "2000"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/local-restic
  uid: 617e3487-5bfb-11e7-bb52-08002711f4aa
spec:
  selector:
    matchLabels:
      app: local-restic
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
  backend:
    local:
      path: /repo
      volumeSource:
        emptyDir: {}
    storageSecretName: local-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```


### AWS S3
Stash supports AWS S3 service or [Minio](https://minio.io/) servers as backend. To configure this backend, following secret keys are needed:

| Key                     | Description                                                |
|-------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AWS_ACCESS_KEY_ID`     | `Required`. AWS / Minio access key ID                      |
| `AWS_SECRET_ACCESS_KEY` | `Required`. AWS / Minio secret access key                  |

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-aws-access-key-id-here>' > AWS_ACCESS_KEY_ID
$ echo -n '<your-aws-secret-access-key-here>' > AWS_SECRET_ACCESS_KEY
$ kubectl create secret generic s3-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./AWS_ACCESS_KEY_ID \
    --from-file=./AWS_SECRET_ACCESS_KEY
secret "s3-secret" created
```

```yaml
$ kubectl get secret s3-secret -o yaml

apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: PHlvdXItYXdzLWFjY2Vzcy1rZXktaWQtaGVyZT4=
  AWS_SECRET_ACCESS_KEY: PHlvdXItYXdzLXNlY3JldC1hY2Nlc3Mta2V5LWhlcmU+
  RESTIC_PASSWORD: Y2hhbmdlaXQ=
kind: Secret
metadata:
  creationTimestamp: 2017-06-28T12:22:33Z
  name: s3-secret
  namespace: default
  resourceVersion: "2511"
  selfLink: /api/v1/namespaces/default/secrets/s3-secret
  uid: 766d78bf-5bfc-11e7-bb52-08002711f4aa
type: Opaque
```

Now, you can create a Restic tpr using this secret. Following parameters are available for `S3` backend.

| Parameter     | Description                                                                     |
|---------------|---------------------------------------------------------------------------------|
| `s3.endpoint` | `Required`. For S3, use `s3.amazonaws.com`. If your bucket is in a different location, S3 server (s3.amazonaws.com) will redirect restic to the correct endpoint. For an S3-compatible server that is not Amazon (like Minio), or is only available via HTTP, you can specify the endpoint like this: `http://server:port`. |
| `s3.bucket`   | `Required`. Name of Bucket. If the bucket does not exist yet it will be created in the default location (`us-east-1` for S3). It is not possible at the moment to have restic create a new bucket in a different location, so you need to create it using a different program.        |
| `s3.prefix`   | `Optional`. Path prefix into bucket where repository will be created.           |

```console
$ kubectl create -f ./docs/examples/backends/s3/s3-restic.yaml
restic "s3-restic" created
```

```yaml
$ kubectl get restic s3-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T12:58:10Z
  name: s3-restic
  namespace: default
  resourceVersion: "4889"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/s3-restic
  uid: 7036ba69-5c01-11e7-bb52-08002711f4aa
spec:
  selector:
    matchLabels:
      app: s3-restic
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
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
```


### Google Cloud Storage (GCS)
Stash supports Google Cloud Storage(GCS) as backend. To configure this backend, following secret keys are needed:

| Key                               | Description                                                |
|-----------------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`                 | `Required`. Password used to encrypt snapshots by `restic` |
| `GOOGLE_PROJECT_ID`               | `Required`. Google Cloud project ID                        |
| `GOOGLE_SERVICE_ACCOUNT_JSON_KEY` | `Required`. Google Cloud service account json key          |

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

Now, you can create a Restic tpr using this secret. Following parameters are available for `gcs` backend.

| Parameter      | Description                                                                     |
|----------------|---------------------------------------------------------------------------------|
| `gcs.bucket`   | `Required`. Name of Bucket. If the bucket does not exist yet, it will be created in the default location (US). It is not possible at the moment to have restic create a new bucket in a different location, so you need to create it using a different program.        |
| `gcs.prefix`   | `Optional`. Path prefix into bucket where repository will be created.           |

```console
$ kubectl create -f ./docs/examples/backends/gcs/gcs-restic.yaml
restic "gcs-restic" created
```

```yaml
$ kubectl get restic gcs-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T13:11:43Z
  name: gcs-restic
  namespace: default
  resourceVersion: "5781"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/gcs-restic
  uid: 54b1bad3-5c03-11e7-bb52-08002711f4aa
spec:
  selector:
    matchLabels:
      app: gcs-restic
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
  backend:
    gcs:
      bucket: stash-qa
      prefix: demo
    storageSecretName: gcs-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```



### Microsoft Azure Storage
Stash supports Microsoft Azure Storage as backend. To configure this backend, following secret keys are needed:

| Key                     | Description                                                |
|-------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AZURE_ACCOUNT_NAME`    | `Required`. Azure Storage account name                     |
| `AZURE_ACCOUNT_KEY`     | `Required`. Azure Storage account key                      |

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

Now, you can create a Restic tpr using this secret. Following parameters are available for `Azure` backend.

| Parameter     | Description                                                                     |
|---------------|---------------------------------------------------------------------------------|
| `azure.container` | `Required`. Name of Storage container                                       |
| `azure.prefix`    | `Optional`. Path prefix into bucket where repository will be created.       |

```console
$ kubectl create -f ./docs/examples/backends/azure/azure-restic.yaml
restic "azure-restic" created
```

```yaml
$ kubectl get restic azure-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T13:31:14Z
  name: azure-restic
  namespace: default
  resourceVersion: "7070"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/azure-restic
  uid: 0e8eb89b-5c06-11e7-bb52-08002711f4aa
spec:
  selector:
    matchLabels:
      app: azure-restic
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
  backend:
    azure:
      container: stashqa
      prefix: demo
    storageSecretName: azure-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```


### OpenStack Swift
Stash supports [OpenStack Swift as backend](https://restic.readthedocs.io/en/stable/manual.html#openstack-swift). To configure this backend, following secret keys are needed:

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

Now, you can create a Restic tpr using this secret. Following parameters are available for `Swift` backend.

| Parameter         | Description                                                                 |
|-------------------|-----------------------------------------------------------------------------|
| `swift.container` | `Required`. Name of Storage container                                       |
| `swift.prefix`    | `Optional`. Path prefix into bucket where repository will be created.       |

```console
$ kubectl create -f ./docs/examples/backends/swift/swift-restic.yaml
restic "swift-restic" created
```

```yaml
$ kubectl get restic swift-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  creationTimestamp: 2017-06-28T13:31:14Z
  name: swift-restic
  namespace: default
  resourceVersion: "7070"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/swift-restic
  uid: 0e8eb89b-5c06-11e7-bb52-08002711f4aa
spec:
  selector:
    matchLabels:
      app: swift-restic
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
  backend:
    swift:
      container: stashqa
      prefix: demo
    storageSecretName: swift-secret
  schedule: '@every 1m'
  volumeMounts:
  - mountPath: /source/data
    name: source-data
```

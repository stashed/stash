# Stash Backends
Backend is where `restic` stores snapshots. For any backend, a Kubernetes Secret in the same namespace is needed to provide restic repository credentials. This Secret can be configured by setting `spec.backend.repositorySecretName` field. This document lists the various supported backends for Stash and how to configure those.

## Local
`Local` backend refers to a local path inside `stash` sidecar container. Any Kubernetes supported [persistent volume](https://kubernetes.io/docs/concepts/storage/volumes/) can be used here. Some examples are: `emptyDir` for testing, NFS, Ceph, GlusterFS, etc. To configure this backend, following secret keys are needed:

| Key               | Description                                                |
|-------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD` | `Required`. Password used to encrypt snapshots by `restic` |

```sh
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

Now, you can create a Restic tpr using this secret. Following parameters are availble for `Local` backend.

| Parameter      | Description                                                                                 |
|----------------|---------------------------------------------------------------------------------------------|
| `local.path`   | `Required`. Path where this volume will be mounted in the sidecar container. Example: /repo |
| `local.volume` | `Required`. Any Kubernetes volume                                                           |

```sh
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
  backend:
    local:
      path: /repo
      volume:
        emptyDir: {}
        name: repo
    repositorySecretName: local-secret
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: local-restic
```


# AWS S3
Stash supports AWS S3 service or [Minio](https://minio.io/) servers as backend. To configure this backend, following secret keys are needed:

| Key                     | Description                                                |
|-------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AWS_ACCESS_KEY_ID`     | `Required`. AWS / Minio access key ID                      |
| `AWS_SECRET_ACCESS_KEY` | `Required`. AWS / Minio secret access key                  |

```sh
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

Now, you can create a Restic tpr using this secret. Following parameters are availble for `S3` backend.

| Parameter     | Description                                                                     |
|---------------|---------------------------------------------------------------------------------|
| `s3.endpoint` | `Required`. For S3, use `s3.amazonaws.com`. If your bucket is in a different location, S3 server (s3.amazonaws.com) will redirect restic to the correct endpoint. For an S3-compatible server that is not Amazon (like Minio), or is only available via HTTP, you can specify the endpoint like this: `http://server:port`. |
| `s3.bucket`   | `Required`. Name of Bucket                                                      |
| `s3.prefix`   | `Optional`. Path prefix into bucket where repository will be created.           |

```sh
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
  backend:
    repositorySecretName: s3-secret
    s3:
      bucket: stash-qa
      endpoint: s3.amazonaws.com
      prefix: demo
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: s3-restic
```


# Google Cloud Storage (GCS)
Stash supports Google Cloud Storage(GCS) as backend. To configure this backend, following secret keys are needed:

| Key                               | Description                                                |
|-----------------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`                 | `Required`. Password used to encrypt snapshots by `restic` |
| `GOOGLE_PROJECT_ID`               | `Required`. Google Cloud project ID                        |
| `GOOGLE_SERVICE_ACCOUNT_JSON_KEY` | `Required`. Google Cloud service account json key          |

```sh
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
  GOOGLE_SERVICE_ACCOUNT_JSON_KEY: ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAicmVzdGljLXFhIiwKICAicHJpdmF0ZV9rZXlfaWQiOiAiNDJlMGMzMTk3NGE4NzRiYmJkMzNmNjQxNDBkOTI0Y2U5ZjkyZTM4MyIsCiAgInByaXZhdGVfa2V5IjogIi0tLS0tQkVHSU4gUFJJVkFURSBLRVktLS0tLVxuTUlJRXZRSUJBREFOQmdrcWhraUc5dzBCQVFFRkFBU0NCS2N3Z2dTakFnRUFBb0lCQVFDd205NGx0eHNSd0QvelxuQi82TC8yNVJQMXdlUWJSeTBVUS8xUGRjbnUrSlMzQnBOb0ZoUEJDYWdzV3loSFJwUzVpN011WDJXOVNlaVUrNFxudHpZcXZmeDFpSTAzNUZYVmhaOXhvb0ZpYjdvYTMxclJaUTY1STc4bGxoQ1hyQXNmdFZ0VWRhQmM2cWV6c0FlUFxuNDJwUGJRb2V4WDFsU2ZmS0tiNzl6UUJiM2dxaERadmxybDMvRzdqaW5pVGhFTUx3YWxjQzZSRGFLQXNiQUIveVxuK09LaTQ5dTkzYlRmdFplaERwY1dBVVQ3S3lqdUd5Z0tGQmdhYk1JVjNHSUpBaGxnV1I4TC9sMm41YWQ0WkE5NlxuK3UrSUpqdExyTkdlOTlwbHJURWcwRFQyNkNGYXRiMnlwWmJQQ1I0b1dzUmprZENjZ0hBWXRIWU9weWIzRnZodFxuNGd2ZzIyNUhBZ01CQUFFQ2dnRUFCcVhuWjk0THE5QmoxOTh1S3REenN5VkNiM1VqdU1xOTJmVkhWbm81SkI3dFxuM1ZnSzZNRWRFdVBuVXovL0xkT0ZyVTVPTDhibkt3eWFMcWJlNkI3OHVPUHFCUGVZYjVBM0gwenh0K1hpeUk0dFxuMmdJRzJ0dEluNzZWWTFBN252Ynh1QzB4V3k0T0lBcDVUbVpPSXkxRW0wSHQ1WGt5VmE3YW5LMHgzVU52ZlA1NFxuOFkwSzdySy8vWkZSQ2lPb25LY1NvamdZTS9uY2R5cjNLM2hpTUN3bWh1dHZoTFRjSG9lWDk4RURjYnNlU0ZCNlxuRXkvOHpxYzNvN1VTUWUxUDVxb1hkdVRxNU5Nb0tCdkJudmVxeG0xbGx2cDRiMEl1Y3lmZkNlazNuS0lndm03N1xucUJUVUdIaktQLzcvOGdGSDNFd2J2VC83WUV5NTVpK0xhVFpyS2tGNm9RS0JnUUQ0LzVTVjNnMEw5bkZjZ3VPV1xuMWQ0a3FPSTJHWmNCc1l3dWp4K0ZrdFBzRUM1L0FVTXMxRlhKcTR2MGVvZHAxbW1sd0VCamlGVC9vSy95Y05hYlxuWklVTGpFeC9IM1RoS3U2QVdiSmdjbU94OGYvTU40SXhOU1Y1QW1DUnB5SEg2UUdTb3Fkc0c4a0g3dWZ1SjlJdVxucnM2MExyZ2JBdWs0M1lmY0V4M0hUODE5VVFLQmdRQzFrekNxS3FNWk9haTVlME5ub3A5ZFNEVGliSUdpRGVJY1xuZzFaeENGa2dCeU5JaUdIdmxhWmlMNFRiL3Q1NmowTFpsL1dJNjc5UlFMSXk0NmU0N1dlNko0MzhweWF2VjRpdVxuU2lPS01yR2V5cXpGVFhRSDVYaW5CUHhtTk96cG9EaVJyeUFVeGd0bWtIeEpOYlRhdS8zbHNpL0Nyc0RyTkJQNlxuVitENURsZHNGd0tCZ1FEVmVYRnZKNGV4K09CcHV3SGFZSk5xaEt3a1M3NHVRb1QzcWRjUmtyZEVEUDkvL1pvVlxuQmhwaW8wT0RIOFdXMUsrUTNvbVZpOTJycDUwUlV2SjdHU3dEb1k0MzhzVW5Bc0tsb2NFUGRTTEovYnNiMzM4c1xuSnU5d2xyd3FROHJ2ZEhIWHdNR2ZLeGNvU1FmcEk1VE1WeXg2U0ErcGdNNW81V3pFSGxPS2ZIMmxjUUtCZ0FuaVxucVpPYUhxY1E3STZzbDA3ZEc3QUlibGlsYjZsUytDeDFPZytOVk16WmxxSXNTcWl3alE1clorQlNUK3A4UWpkMlxuZm5lbDNoU2VZUlZFTDYxeHYyUHpJMWZPQWQwcDl0Y0dVa2tEMlllN29ReGMyeVJTNmU2dDVzL3BzYnhHYk00QlxucXMxMnVzZ3F0Wm1Hd3dIbG1qMFhKbUtEQVIzTkNBbHBIMlp2MFhLaEFvR0FQMXh4WFFnbHRuc2RZYjRQK0RHTVxuV3VlSXI5Yys0dWxjYTNWdmZna0FGa1RoOWdrN3E3WDJGK0Y5Z1lscVQyK1h5RkphMnRlREhKOUdSME5Da09iNlxuWTVPSk1UTkREUU9kSXdzS1VvZVg4d0p4YVhXSmpxWFBGWDZ0djBkTlpVME5jbzNBMFV2ZjlOd25CTW0vUDZMTFxuclRGS0M5bmNqOWVVNTFTUXdmaW45K3M9XG4tLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tXG4iLAogICJjbGllbnRfZW1haWwiOiAicmVzdGljLXRlc3RpbmdAcmVzdGljLXFhLmlhbS5nc2VydmljZWFjY291bnQuY29tIiwKICAiY2xpZW50X2lkIjogIjEwMjY5NTk1MzE0NzQzNDk1ODk1MSIsCiAgImF1dGhfdXJpIjogImh0dHBzOi8vYWNjb3VudHMuZ29vZ2xlLmNvbS9vL29hdXRoMi9hdXRoIiwKICAidG9rZW5fdXJpIjogImh0dHBzOi8vYWNjb3VudHMuZ29vZ2xlLmNvbS9vL29hdXRoMi90b2tlbiIsCiAgImF1dGhfcHJvdmlkZXJfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHMiLAogICJjbGllbnRfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9yb2JvdC92MS9tZXRhZGF0YS94NTA5L3Jlc3RpYy10ZXN0aW5nJTQwcmVzdGljLXFhLmlhbS5nc2VydmljZWFjY291bnQuY29tIgp9Cg==
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

Now, you can create a Restic tpr using this secret. Following parameters are availble for `gcs` backend.

| Parameter      | Description                                                                     |
|----------------|---------------------------------------------------------------------------------|
| `gcs.location` | `Required`. Name of Google Cloud region.                                        |
| `gcs.bucket`   | `Required`. Name of Bucket                                                      |
| `gcs.prefix`   | `Optional`. Path prefix into bucket where repository will be created.           |

```sh
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
  backend:
    gcs:
      bucket: stash-qa
      location: /repo
      prefix: demo
    repositorySecretName: gcs-secret
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: gcs-restic
```


# Microsoft Azure Storage
Stash supports Microsoft Azure Storage as backend. To configure this backend, following secret keys are needed:

| Key                     | Description                                                |
|-------------------------|------------------------------------------------------------|
| `RESTIC_PASSWORD`       | `Required`. Password used to encrypt snapshots by `restic` |
| `AZURE_ACCOUNT_NAME`    | `Required`. Azure Storage account name                     |
| `AZURE_ACCOUNT_KEY`     | `Required`. Azure Storage account key                      |

```sh
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

Now, you can create a Restic tpr using this secret. Following parameters are availble for `Azure` backend.

| Parameter     | Description                                                                     |
|---------------|---------------------------------------------------------------------------------|
| `azure.container` | `Required`. Name of Storage container                                       |
| `azure.prefix`    | `Optional`. Path prefix into bucket where repository will be created.       |

```sh
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
  backend:
    azure:
      container: stashqa
      prefix: demo
    repositorySecretName: azure-secret
  fileGroups:
  - path: /lib
    retentionPolicy:
      keepLastSnapshots: 5
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: azure-restic
```

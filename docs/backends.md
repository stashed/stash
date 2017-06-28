# Stash Backends
Backend are where snapshots stored by `restic`. This document lists the various supported backends for Stash and how to configure those.

## Local
`Local` backend refers to a local path inside `stash` sidecar container. Any Kubernetes supported [persistent volume](https://kubernetes.io/docs/concepts/storage/volumes/) can be used here. Some examples are: `emptyDir` for testing, NFS, Ceph, GlusterFS, etc. To configure this backend, following secret keys are needed:

| Key               | Description                                    |
|-------------------|------------------------------------------------|
| `RESTIC_PASSWORD` | Password used to encrypt snapshots by `restic` |

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
| Parameter      |                                                                                 |
|----------------|---------------------------------------------------------------------------------|
| `local.path`   | Path where this volume will be mounted in the sidecar container. Example: /repo |
| `local.volume` | Any Kubernetes volume                                                           |

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

| Key                     | Description                                    |
|-------------------------|------------------------------------------------|
| `RESTIC_PASSWORD`       | Password used to encrypt snapshots by `restic` |
| `AWS_ACCESS_KEY_ID`     | AWS access key ID                              |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key                          |

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

## 


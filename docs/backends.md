# Stash Backends
Backend are where snapshots stored by `restic`. This document lists the various supported backends for Stash and how to configure those.

## Local
`Local` backend refers to a local path inside `stash` sidecar container. Any Kubernetes supported [persistent volume](https://kubernetes.io/docs/concepts/storage/volumes/) can be used here. Some examples are: `emptyDir` for testing, NFS, Ceph, GlusterFS, etc. To configure this backend, following secret keys are needed:

| Key               | Description                                    |
|-------------------|------------------------------------------------|
| `RESTIC_PASSWORD` | Password used to encrypt snapshots by `restic` |





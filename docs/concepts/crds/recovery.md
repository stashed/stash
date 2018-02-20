---
title: Recovery Overview
menu:
  product_stash_0.7.0-rc.0:
    identifier: recovery-overview
    name: Recovery
    parent: crds
    weight: 15
product_name: stash
menu_name: product_stash_0.7.0-rc.0
section_menu_id: concepts
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Recoveries

## What is Recovery
A `Recovery` is a Kubernetes `CustomResourceDefinition` (CRD). It provides configuration for restoring a backup taken using Stash. You only need to specify the `Restic` that was used for taking backup, the target workload and volume where backup will be restored.

## Recovery Spec
As with all other Kubernetes objects, a Recovery needs `apiVersion`, `kind`, and `metadata` fields. It also needs a `.spec` section. Below is an example Recovery object.

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: stash-demo
  namespace: default
spec:
  workload:
    kind: Deployment
    name: stash-demo
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
```

The `.spec` section has following parts:

### spec.workload
`spec.workload` specifies a target workload that was backed up using `Restic`. A single `Restic` backups all types of workloads that matches the label-selector, but you can only restore a specific workload using a `Recovery`.

### spec.podOrdinal
For workload kind `Statefulset`, you need to specify pod [index](https://kubernetes.io/docs/guides/stateful-application/basic-stateful-set/#pods-in-a-statefulset) using `spec.podOrdinal`. You must not specify it for other workload kinds. For example:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: statefulset-demo
  namespace: default
spec:
  workload:
    kind: Statefulset
    name: statefulset-demo
  podOrdinal: 0
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
```

### spec.nodeName
For workload kind `Daemonset`, you need to specify node name using `spec.nodeName`. You must not specify it for other workload kinds. For example:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: daemonset-demo
  namespace: default
spec:
  restic: daemonset-demo
  workload:
    kind: Daemonset
    name: daemonset-demo
  nodeName: minikube
  backend:
    local:
      mountPath: /safe/data
      hostPath:
        path: /data/stash-test/restic-repo
    storageSecretName: stash-demo
  paths:
  - /source/data
  recoveredVolumes:
  - mountPath: /source/data
    hostPath:
      path: /data/stash-test/restic-restored
```

### spec.backend
Specifies the backend that was used in `Restic` to take backups.
To learn how to configure various backends for Restic, please visit [here](/docs/guides/backends.md).

### spec.paths
Array of strings specifying the file-group paths that was backed up using `Restic`.

### spec.recoveredVolumes
Indicates an array of volumes where snapshots will be recovered. Here, `path` specifies where the volume will be mounted.
Note that, `Recovery` recovers data in the same paths from where backup was taken (specified in `spec.paths`). So, volumes must be mounted on those paths or their parent paths.
Following parameters are available for `recoveredVolumes`.

| Parameter                       | Description                                                                                   |
|---------------------------------|-----------------------------------------------------------------------------------------------|
| `recoveredVolumes.mountPath`    | `Required`. Path where this volume will be mounted in the sidecar container. Example: `/repo` |
| `recoveredVolumes.subPath`      | `Optional`. Sub-path inside the referenced volume instead of its root.                        |
| `recoveredVolumes.VolumeSource` | `Required`. Any Kubernetes volume. Can be specified inlined. Example: `hostPath`

## Recovery Status

Stash operator updates `.status` of a Recovery CRD when recovery operation is completed.

 - `status.phase` indicates the current phase of overall recovery process. Possible values are `Pending`, `Running`, `Succeeded`, `Failed` and `Unknown`.
 - `status.stats` is a array status, each of which indicates the status for individual paths. Each element of the array has following fields:
   - `status.stats[].path` indicates a path that was backed up using `Restic` and is selected for recovery.
   - `status.stats[].phase` indicates the current phase of recovery process for the particular path. Possible values are `Pending`, `Running`, `Succeeded`, `Failed` and `Unknown`.
   - `status.stats[].duration` indicates the elapsed time to successfully restore backup for the particular path.

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- To restore a backup see [here](/docs/guides/restore.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/guides/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Learn about how to configure Stash operator as workload initializer [here](/docs/guides/initializer.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

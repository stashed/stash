---
title: Concept | Stash
description: Concepts of Stash
menu:
  product_stash_0.5.1:
    identifier: concept-stash
    name: Concept
    parent: getting-started
    weight: 30
product_name: stash
menu_name: product_stash_0.5.1
section_menu_id: getting-started
url: /products/stash/0.5.1/getting-started/concept/
aliases:
  - /products/stash/0.5.1/concept/
---

> New to Stash? Please start [here](/docs/tutorials/README.md).

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
  restic: stash-demo
  workload:
    kind: Deployment
    name: stash-demo
  volumes:
  - name: restored-data
    hostPath:
      path: /data/stash-test/restic-restored
```

The `.spec` section has following parts:

### spec.restic
`spec.restic` specifies the `Restic` name that was used to take backups.

### spec.workload
`spec.workload` specifies a target workload that was backed up using `Restic`. A single `Restic` backups all types of workloads that matches the label-selector, but you can only restore a specific workload using a `Recovery`. 

### spec.podOrdinal
For workload kind `Statefulset`, you need to specify pod [index](https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#pods-in-a-statefulset) using `spec.podOrdinal`. You must not specify it for other workload kinds. For example:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: statefulset-demo
  namespace: default
spec:
  restic: statefulset-demo
  workload:
    kind: Statefulset
    name: statefulset-demo
  podOrdinal: 0
  volumes:
  - name: restored-data
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
  volumes:
  - name: restored-data
    hostPath:
      path: /data/stash-test/restic-restored
```

### spec.volumes
`spec.volumes` indicates an array of volumes where snapshots will be recovered. Here, `volume.name` should be same as the workload volume name that was backed up using `Restic`.

## Recovery Status

Stash operator updates `.status` of a Recovery CRD when recovery operation is completed. 

 - `status.phase` indicates the current phase of overall recovery process. Possible values are `Pending`, `Running`, `Succeeded`, `Failed` and `Unknown`.
 - `status.stats` is a array status, each of which indicates the status for individual paths. Each element of the array has following fields:
   - `status.stats[].path` indicates a path that was backed up using `Restic` and is selected for recovery.
   - `status.stats[].phase` indicates the current phase of recovery process for the particular path. Possible values are `Pending`, `Running`, `Succeeded`, `Failed` and `Unknown`.
   - `status.stats[].duration` indicates the elapsed time to successfully restore backup for the particular path.

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/tutorials/backup.md).
- Learn about the details of Restic CRD [here](/docs/concept_restic.md).
- To restore a backup see [here](/docs/tutorials/restore.md).
- To run backup in offline mode see [here](/docs/tutorials/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/tutorials/backends.md).
- See working examples for supported workload types [here](/docs/tutorials/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/tutorials/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/tutorials/rbac.md).
- Learn about how to configure Stash operator as workload initializer [here](/docs/tutorials/initializer.md).
- Wondering what features are coming next? Please visit [here](/ROADMAP.md). 
- Want to hack on Stash? Check our [contribution guidelines](/CONTRIBUTING.md).

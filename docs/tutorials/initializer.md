> New to Stash? Please start [here](/docs/tutorials/README.md).

## Workload Initializer

Stash operator can be used as a workload [initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/#initializers). For this you need to create a `InitializerConfiguration` with initializer named `stash.appscode.com`.

```console
$ kubectl apply -f ./hack/deploy/initializer.yaml
initializerconfiguration "stash-initializer-config" created
```

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: InitializerConfiguration
metadata:
  labels:
    app: stash
  name: stash-initializer
initializers:
- name: stash.appscode.com
  rules:
  - apiGroups:
    - "*"
    apiVersions:
    - "*"
    resources:
    - daemonsets
    - deployments
    - replicasets
    - replicationcontrollers
    - statefulsets
```

This is helpful when you create `Restic` before creating workload objects. This allows stash operator to initialize the target workloads by adding sidecar or, init-container before workload-pods are created. Thus stash operator do not need to delete workload pods for applying changes.

This is particularly helpful for workload kind `StatefulSet` since kubernetes does not support updating StatefulSet after they are created.

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/tutorials/backup.md).
- Learn about the details of Restic CRD [here](/docs/concept_restic.md).
- To restore a backup see [here](/docs/tutorials/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concept_recovery.md).
- To run backup in offline mode see [here](/docs/tutorials/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/tutorials/backends.md).
- See working examples for supported workload types [here](/docs/tutorials/workloads.md).
- Thinking about monitoring your backup operations? Stash works [out-of-the-box with Prometheus](/docs/tutorials/monitoring.md).
- Learn about how to configure [RBAC roles](/docs/tutorials/rbac.md).
- Wondering what features are coming next? Please visit [here](/ROADMAP.md).
- Want to hack on Stash? Check our [contribution guidelines](/CONTRIBUTING.md).
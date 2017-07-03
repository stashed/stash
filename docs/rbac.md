# Configuring RBAC

To use Stash in a cluster with RBAC enabled, [install Stash](/docs/install.md) with RBAC options.

Sidecar container added to workloads makes various calls to Kubernetes api. ServiceAccounts used with workloads should have the following roles:

```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: stash-sidecar
rules:
- apiGroups:
  - stash.appscode.com
  resources: ["*"]
  verbs: ["*"]
- apiGroups:
  - extensions
  resources:
  - deployments
  - daemonsets
  - replicasets
  verbs: ["get"]
- apiGroups: [""]
  resources:
  - replicationcontrollers
  verbs: ["*"]
- apiGroups: [""]
  resources:
  - secrets
  verbs: ["get"]
- apiGroups: [""]
  resources:
  - events
  verbs: ["create"]
```

Further discussion is ongoing whether Stash should automatically configure RBAC for workload service accounts. Please give your feedback [here](https://github.com/appscode/stash/issues/123).

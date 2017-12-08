

## Upgrade Steps

1. Backup all your old `Restic` CRDs.
2. Delete your `Restic` objects. It will stop taking backups and remove sidecars from pods.
3. Uninstall old `Stash` operator.
4. Move repositories to new location if you want to keep your old backups.
5. Install new `Stash` operator.
6. Update old `Restic` CRDs and deploy them. It will add sidecar to pods and continue backup.

## Repository Location

We have changed repository location in new version of `Stash` to remove conflicts between  repositories for different target workloads.
 
### Old Version

```
Location depending on restic.spec.useAutoPrefix:

Not Specified or, Smart:

    Deployment:             {BackendPrefix}/
    Replica Set:            {BackendPrefix}/
    Replication Controller: {BackendPrefix}/
    Stateful Set:           {BackendPrefix}/{PodName}/
    Daemon Set:             {BackendPrefix}/{NodeName}/

NodeName: {BackendPrefix}/{NodeName}/
PodName:  {BackendPrefix}/{PodName}/
None:     {BackendPrefix}/
```

### New Version

```
Deployment:             {BackendPrefix}/deployment/{WorkloadName}/
Replica Set:            {BackendPrefix}/replicaset/{WorkloadName}/
Replication Controller: {BackendPrefix}/replicationcontroller/{WorkloadName}/
Stateful Set:           {BackendPrefix}/statefulset/{PodName}/
Daemon Set:             {BackendPrefix}/daemonset/{WorkloadName}/{NodeName}/
```

## S3 Example

Consider you have following old `Restic` CRD:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: s3-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo
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
Let say your cluster has two nodes named `node-1` and `node-2` and you have following workloads with `app=stash-demo` label. For older version of `Stash`, backup will be created in following locations:

| Workload Kind | Workload Name | Replicas | Old Location |
| --- | --- | --- | --- |
| Deployment | my-deploy | 2 | stash-qa/demo |
| Satefulset | my-satefulset | 2 | stash-qa/demo/my-satefulset-0 <br> stash-qa/demo/my-satefulset-1 |
| Daemonset | my-daemonset | - | stash-qa/demo/node-1 <br> stash-qa/demo/node-2 |

### Step 1

Backup your old `Restic` CRD.

```
$ kubectl get restic s3-restic -o yaml --export > s3-restic.yaml
```

### Step 2

Delete the old`Restic` object.

```
$ kubectl delete restic s3-restic
```

### Step 3

Uninstall old `Stash` operator by following [this]().

### Step 4

To keep your old backups you should move the contents of your old repositories to new locations.

| Workload Kind | Workload Name | Replicas | New Location |
| --- | --- | --- | --- |
| Deployment | my-deploy | 2 | stash-qa/demo/deployment/my-deploy |
| Satefulset | my-satefulset | 2 | stash-qa/demo/satefulset/my-satefulset-0 <br> stash-qa/demo/satefulset/my-satefulset-1 |
| Daemonset | my-daemonset | - | stash-qa/demo/daemonset/my-daemonset/node-1 <br> stash-qa/demo/daemonset/my-daemonset/node-2 |

### Step 5

Install new `Stash` operator by folling [this]().

### Step 6

Now update your old `Restic` CRD as follows:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: s3-restic
  namespace: default
spec:
  selector:
    matchLabels:
      app: stash-demo
  fileGroups:
  - path: /source/data
    retentionPolicyName: policy-1
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
  retentionPolicies:
      name: policy-1
      keepLast: 5
      prune: true
```

Now deploy your new `Restic` CRD.

```
$ kubectl create -f s3-restic.yaml
```

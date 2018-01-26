---
title: Upgrade | Stash
description: Stash Upgrade
menu:
  product_stash_0.7.0-alpha.0:
    identifier: upgrade-stash
    name: Upgrade
    parent: setup
    weight: 15
product_name: stash
menu_name: product_stash_0.7.0-alpha.0
section_menu_id: setup
---

# Upgrading Stash

If you are upgrading Stash to a patch release, please reapply the [installation instructions](/docs/setup/install.md). That will upgrade the operator pod to the new version and fix any RBAC issues.

## Upgrading from 0.5.x to 0.6.x

The format for `Restic` object has changed in backward incompatiable manner between 0.5.x and 0.7.0-alpha.0 . The steps involved in upgrading Stash operator to 0.7.0-alpha.0 from prior version involves the following steps:

1. Backup all your old `Restic` CRDs.
2. Delete your old `Restic` objects. It will stop taking backups and remove sidecars from pods.
3. Uninstall old `Stash` operator.
4. Move repositories to new location if you want to keep your old backups. We have changed repository location in new version of `Stash` to remove conflicts between  repositories for different target workloads.

**Old Version**

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

**New Version**

```
Deployment:             {BackendPrefix}/deployment/{WorkloadName}/
Replica Set:            {BackendPrefix}/replicaset/{WorkloadName}/
Replication Controller: {BackendPrefix}/replicationcontroller/{WorkloadName}/
Stateful Set:           {BackendPrefix}/statefulset/{PodName}/
Daemon Set:             {BackendPrefix}/daemonset/{WorkloadName}/{NodeName}/
```

5. Install new `Stash` operator.
6. Update your backed up `Restic` CRDs in the following ways:
- `restic.spec.useAutoPrefix` is removed in new version of `Stash`. If you specify it in your old `Restic` CRD, you should remove it.
- `restic.spec.fileGroups[].retentionPolicy` is moved to `restic.spec.retentionPolicies[]` and referenced using `restic.spec.fileGroups[].retentionPolicyName`.

Consider following example:

**Old version:**

```yaml
fileGroups:
- path: /source/path-1
  retentionPolicy:
    keepLast: 5
    prune: true
- path: /source/path-2
  retentionPolicy:
    keepLast: 10
- path: /source/path-3
  retentionPolicy:
    keepLast: 5
    prune: true
```

**New version:**

```yaml
fileGroups:
- path: /source/path-1
  retentionPolicyName: policy-1
- path: /source/path-2
  retentionPolicyName: policy-2
- path: /source/path-3
  retentionPolicyName: policy-1
retentionPolicies:
- name: policy-1
  keepLast: 5
  prune: true
- name: policy-2
  keepLast: 10
```

7. Now, re-deploy restic CRDs. It will add sidecar to pods again and continue backup.


### Example: Upgrading S3 backed volume

Say, you are running `Stash` operator `0.5.1`.

```console
$ kubectl get pods --all-namespaces -l app=stash
NAMESPACE     NAME                              READY     STATUS    RESTARTS   AGE
kube-system   stash-operator-7cdc467c5b-drj2r   2/2       Running   0          2s

$ kubectl exec -it stash-operator-7cdc467c5b-drj2r -c operator -n kube-system stash version

Version = 0.5.1
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = f87995af4875d5e99978def17186ac1957871c1d
GitBranch = release-0.5
GitTag = 0.5.1
CommitTimestamp = 2017-10-10T17:05:31
```

Consider you have following `busybox` deployment:

```yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  labels:
    app: stash-demo
  name: stash-demo
  namespace: default
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - command:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      restartPolicy: Always
      volumes:
      - gitRepo:
          repository: https://github.com/appscode/stash-data.git
        name: source-data
```

Let say you are using following old `Restic` CRD to backup the `busybox` deployment:

```yaml
$ kubectl get restic s3-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  clusterName: ""
  creationTimestamp: 2017-12-11T08:29:16Z
  deletionGracePeriodSeconds: null
  deletionTimestamp: null
  generation: 0
  initializers: null
  name: s3-restic
  namespace: default
  resourceVersion: "39256"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/s3-restic
  uid: 5ffb03c7-de4d-11e7-af32-0800274b5eab
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: s3.amazonaws.com
      prefix: demo
    storageSecretName: s3-secret
  fileGroups:
  - path: /source/data
    retentionPolicy:
      keepLast: 5
      prune: true
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
  volumeMounts:
  - mountPath: /source/data
    name: source-data
status:
  backupCount: 3
  firstBackupTime: 2017-12-11T08:30:40Z
  lastBackupDuration: 19.968827042s
  lastBackupTime: 2017-12-11T08:32:40Z
```

A restic-repository is initialized in `demo` folder of your `stash-qa` bucket and so far 3 backups are created and stored there. Now, the following steps will upgrade version of `Stash` without loosing those 3 backups.

#### Step 1

Dump your old `Restic` CRD in a file.

```
$ kubectl get restic s3-restic -o yaml --export > s3-restic-dump.yaml
```

#### Step 2

Delete the old `Restic` object.

```
$ kubectl delete restic s3-restic
restic "s3-restic" deleted
```

Now wait for sidecar to be removed by `Stash` operator.

```
$ kubectl get pods -l app=stash-demo
NAME                          READY     STATUS    RESTARTS   AGE
stash-demo-6b5459b8d6-7rvpv   2/2       Terminating   0       5s
stash-demo-788ffcf9c6-xh6nx   1/1       Running       0       1s
```

#### Step 3

Uninstall old `Stash` operator by following the instructions [here](/docs/setup/uninstall.md).

#### Step 4

To keep your old backups you should move the contents of your old repositories to new locations.

```
Old location: stash-qa/demo
New location: stash-qa/demo/deployment/stash-demo
```

You can use [aws-cli](https://aws.amazon.com/cli) to do this:

```
$ aws s3 mv s3://stash-qa/demo s3://stash-qa/demo/deployment/stash-demo --recursive
```

#### Step 5

Install new `Stash` operator by following the instructions [here](/docs/setup/install.md).

#### Step 6

Now update your backed up `Restic` CRD as follows:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  clusterName: ""
  creationTimestamp: 2017-12-11T08:29:16Z
  deletionGracePeriodSeconds: null
  deletionTimestamp: null
  generation: 0
  initializers: null
  name: s3-restic
  namespace: default
  resourceVersion: "39256"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/s3-restic
  uid: 5ffb03c7-de4d-11e7-af32-0800274b5eab
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: s3.amazonaws.com
      prefix: demo
    storageSecretName: s3-secret
  fileGroups:
  - path: /source/data
    retentionPolicyName: policy-1
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
  volumeMounts:
  - mountPath: /source/data
    name: source-data
  retentionPolicies:
  - name: policy-1
    keepLast: 5
    prune: true
status:
  backupCount: 3
  firstBackupTime: 2017-12-11T08:30:40Z
  lastBackupDuration: 19.968827042s
  lastBackupTime: 2017-12-11T08:32:40Z
```

Re-deploy the updated `Restic` CRD.

```
$ kubectl apply -f s3-restic.yaml
```

It will add sidecar to the `busybox` pods again and continue backup from where it stopped. You can check status of the `Restic` CRD to verify that new backups are creating.

```yaml
$ kubectl get restic s3-restic -o yaml

apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  clusterName: ""
  creationTimestamp: 2017-12-11T09:36:08Z
  deletionGracePeriodSeconds: null
  deletionTimestamp: null
  generation: 0
  initializers: null
  name: s3-restic
  namespace: default
  resourceVersion: "44000"
  selfLink: /apis/stash.appscode.com/v1alpha1/namespaces/default/restics/s3-restic
  uid: b7511b7a-de56-11e7-af32-0800274b5eab
spec:
  backend:
    s3:
      bucket: stash-qa
      endpoint: s3.amazonaws.com
      prefix: demo
    storageSecretName: s3-secret
  fileGroups:
  - path: /source/data
    retentionPolicyName: policy-1
  retentionPolicies:
  - keepLast: 5
    name: policy-1
    prune: true
  schedule: '@every 1m'
  selector:
    matchLabels:
      app: stash-demo
  volumeMounts:
  - mountPath: /source/data
    name: source-data
status:
  backupCount: 4
  firstBackupTime: 2017-12-11T08:30:40Z
  lastBackupDuration: 18.786502029s
  lastBackupTime: 2017-12-11T09:37:38Z
```


### Example: Upgrading GCS backed volume

Consider you have following gcs backend instead of s3 backend:

```yaml
backend:
  gcs:
    bucket: stash-qa
    prefix: demo
  storageSecretName: gcs-secret
```

You can follow the same steps as the above s3 example. To move old repository to new location using [gsutil](https://cloud.google.com/storage/docs/gsutil/commands/mv#renaming-bucket-subdirectories), run:

```console
$ gsutil mv gs://stash-qa/demo gs://stash-qa/demo/deployment/stash-demo
```

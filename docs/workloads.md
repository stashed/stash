# Supported Workloads

Stash suuports the following types of Kubernetes workloads.

## Deployments
To backup a Deployment, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/deployment.yaml).

## ReplicaSets
To backup a ReplicaSet, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/replicaset.yaml).

## ReplicationControllers
To backup a ReplicationController, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/rc.yaml).

## DaemonSets
To backup a DaemonSet, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/daemonset.yaml). This example shows how Stash can be used to backup host paths on all nodes of a cluster. First run a DaemonSet wihtout nodeSelectors. The only purpose of this DaemonSet to act as an vector for Restic sidecar. In this example, we use a `busybox` container for this. Now, create a Restic that has a fileGroup with path `/srv/host-etc` and `sourceVolumeName`


## StatefulSets


For enabling the backup process for a particular kubernetes object like `RC`, `Replica Set`, `Deployment`, `DaemonSet` user adds a label `restic.appscode.com/config: <name_of_tpr>`. `<name_of_tpr>` is the name of Stash object. And then user creates the Stash object for starting backup process.
In case of StaefulSet user has to add the restic-sidecar container manually.


```yaml
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  labels:
    app: statefulset-demo
  name: workload
  namespace: default
spec:
  replicas: 1
  serviceName: headless
  template:
    metadata:
      labels:
        app: statefulset-demo
      name: busybox
    spec:
      containers:
      - command:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
        resources: {}
      - args:
        - schedule
        - --v=3
        - --namespace=default
        - --name=statefulset-restic
        - --app=workload
        - --prefix-hostname=true
        image: appscode/stash:canary
        imagePullPolicy: IfNotPresent
        name: stash
        volumeMounts:
        - mountPath: /tmp
          name: stash-scratchdir
        - mountPath: /etc
          name: stash-podinfo
        - mountPath: /repo
          name: repo
      restartPolicy: Always
      volumes:
      - emptyDir: {}
        name: stash-scratchdir
      - downwardAPI:
          defaultMode: 420
          items:
          - fieldRef:
              apiVersion: v1
              fieldPath: metadata.labels
            path: labels
        name: stash-podinfo
      - emptyDir: {}
        name: repo
```

You can find the full working example in [examples folder](/docs/examples/workloads/statefulset.yaml).

> New to Stash? Please start [here](/docs/tutorial.md).

# Supported Workloads

Stash supports the following types of Kubernetes workloads.

## Deployments
To backup a Deployment, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/deployment.yaml).

## ReplicaSets
To backup a ReplicaSet, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/replicaset.yaml).

## ReplicationControllers
To backup a ReplicationController, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/rc.yaml).

## DaemonSets
To backup a DaemonSet, create a Restic with matching selectors. You can find the full working demo in [examples folder](/docs/examples/workloads/daemonset.yaml). This example shows how Stash can be used to backup host paths on all nodes of a cluster. First run a DaemonSet without nodeSelectors. The only purpose of this DaemonSet to act as an vector for Restic sidecar. In this example, we use a `busybox` container for this. Now, create a Restic that has a fileGroup with path `/srv/host-etc` and `sourceVolumeName`

## StatefulSets
Kubernetes does not support updating StatefulSet after they are created. So, to backup volumes of a StatefulSet, please add Stash sidecar container to your StatefulSet. You can see the relevant portions of a working example below: 

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
        - --restic-name=statefulset-restic
        - --app=workload
        - --prefix-hostname=true
        image: appscode/stash:0.2.0
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

You can find the full working example in [examples folder](/docs/examples/workloads/statefulset.yaml). To learn about the meaning of various flags, please visit [here](/docs/reference/stash_schedule.md).

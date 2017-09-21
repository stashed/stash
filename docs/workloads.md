> New to Stash? Please start [here](/docs/tutorial.md).

# Supported Workloads

Stash supports the following types of Kubernetes workloads.

## Deployments
To backup a Deployment, create a Restic with matching selectors. You can find a full working demo in [examples folder](/docs/examples/workloads/deployment.yaml).

## ReplicaSets
To backup a ReplicaSet, create a Restic with matching selectors. You can find a full working demo in [examples folder](/docs/examples/workloads/replicaset.yaml).

## ReplicationControllers
To backup a ReplicationController, create a Restic with matching selectors. You can find a full working demo in [examples folder](/docs/examples/workloads/rc.yaml).

## DaemonSets
To backup a DaemonSet, create a Restic with matching selectors. You can find a full working demo in [examples folder](/docs/examples/workloads/daemonset.yaml). This example shows how Stash can be used to backup host paths on all nodes of a cluster. First run a DaemonSet without nodeSelectors. This DaemonSet acts as a vector for Restic sidecar and mounts host paths that are to be backed up. In this example, we use a `busybox` container for this. Now, create a Restic that has a matching selector. This Restic also `spec.volumeMounts` the said host path and points to the host path in `spec.fileGroups`.

## StatefulSets
Kubernetes does not support updating StatefulSet after they are created. So, Stash has limited automated support for StatefulSets. To backup volumes of a StatefulSet, please add Stash sidecar container to your StatefulSet. You can see the relevant portions of a working example below: 

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
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      - args:
        - schedule
        - --restic-name=statefulset-restic
        - --workload=StatefulSet/workload
        - --v=3
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        image: appscode/stash:0.5.0-rc.0
        imagePullPolicy: IfNotPresent
        name: stash
        volumeMounts:
        - mountPath: /tmp
          name: stash-scratchdir
        - mountPath: /etc/stash
          name: stash-podinfo
        - mountPath: /source/data
          name: source-data
          readOnly: true
        - mountPath: /safe/data
          name: stash-local
      restartPolicy: Always
      volumes:
      - gitRepo:
          repository: https://github.com/appscode/stash-data.git
        name: source-data
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
        name: stash-local
```

You can find the full working demo in [examples folder](/docs/examples/workloads/statefulset.yaml). The section you should change for your own StatefulSet are:
 - `--restic-name` flag should be set to the name of the Restic used as configuration.
 - `--workload` flag points to "Kind/Name" of workload where sidecar pod is added. Here are some examples:
   - Deployment: Deployments/xyz, Deployment/xyz, deployments/xyz, deployment/xyz 
   - ReplicaSet: ReplicaSets/xyz, ReplicaSet/xyz, replicasets/xyz, replicaset/xyz, rs/xyz
   - RepliationController: ReplicationControllers/xyz, ReplicationController/xyz, replicationcontrollers/xyz, replicationcontroller/xyz, rc/xyz
   - DaemonSet: DaemonSets/xyz, DaemonSet/xyz, daemonsets/xyz, daemonset/xyz
   - StatefulSet: StatefulSets/xyz, StatefulSet/xyz

To learn about the meaning of various flags, please visit [here](/docs/reference/stash_schedule.md).

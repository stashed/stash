# Supported Workloads

## Deployments, ReplicaSets and ReplicationControllers


## DaemonSets


## StatefulSets

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
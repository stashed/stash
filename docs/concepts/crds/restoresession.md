---
title: RestoreSession Overview
menu:
  product_stash_0.8.3:
    identifier: restoresession-overview
    name: RestoreSession
    parent: crds
    weight: 25
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# RestoreSession

## What is RestoreSession

A `RestoreSession` is a Kubernetes `CustomResourceDefinition (CRD)` which specifies where to restore (target) and the source of data that will be restored in Kubernetes native way.

You have to create a `RestoreSession` object whenever you want to restore. When a `RestoreSession` object is created, Stash injects an `init-container` into the target workload and restart it. The `init-container` restore the desired data. If the target is a database or a stand-alone PVC, Stash lunch a job to perform the restore process.

>In this tutorial, we are going to use **host** word to indicate an entity (pod) where data is restored.

>- For `Deployment`, `ReplicationController` and `ReplicaSet`, restore process run in only one pod. This pod is referred as **host-0**.
>- For `StatefulSet`, restore process runs in all pods. In this case, **pod-0** is known as **host-0**, **pod-1** is known as **host-1**, **pod-2** is known as **host-2** and so on.
>- For `DaemonSet`, restore process runs in all daemon pods. In this case, the **node name** where the pod is running act as their **host** name.
>- For database or stand-alone PVC, restore is done by a job. In this case, the restore job's pod is known as **host-0**.

## RestoreSession CRD Specification

Like other official Kubernetes resources, `RestoreSession` has `TypeMeta`, `ObjectMeta`, `Spec` and `Status` sections.

A sample `RestoreSession` object to restore backed up data of a StatefulSet is shown below,

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: statefulset-restore
  namespace: demo
spec:
  repository:
    name: local-repo
  # task:
  #   name: workload-restore # task field is not required for workload data restore but it is necessary for database restore.
  target:
    ref:
      apiVersion: apps/v1
      kind: StatefulSet
      name: recovered-statefulset
    volumeMounts:
    - mountPath: /source/data
      name: source-data
  rules:
  - subjects: ["host-3","host-4"] # "host-3" and "host-4" will have restored data of backed up host "host-1"
    sourceHost: "host-1" # source host
    paths:
    - /source/data
  - subjects: [] # empty host match all hosts
    sourceHost: "" # no source host indicates that the host is pod itself
    paths:
    - /source/data
  runtimeSettings:
    container:
      resources:
        limits:
          memory: 256M
        requests:
          memory: 256M
      securityContext:
        runAsGroup: 2000
        runAsUser: 2000
      ionice:
        class: 2
        classData: 4
      nice:
        adjustment: 5
    pod:
      imagePullSecrets:
      - name: my-private-registry-secret
      serviceAccountName: my-restore-svc
  tempDir:
    disableCache: false
    medium: Memory
    size: 2Gi
status:
  phase: Succeeded
  totalHosts: 5
  sessionDuration: 4.148288404s
  stats:
  - duration: 884.431745ms
    hostname: host-1
    phase: Succeeded
  - duration: 769.924342ms
    hostname: host-2
    phase: Succeeded
  - duration: 868.694738ms
    hostname: host-3
    phase: Succeeded
  - duration: 792.097784ms
    hostname: host-4
    phase: Succeeded
  - duration: 833.139795ms
    hostname: host-0
    phase: Succeeded
```

Here, we are going to describe some important sections of `RestoreSession` crd.

### RestoreSession `Spec` Section

RestoreSession object holds the following fields in `.spec` section.

#### spec.repository

`spec.repository.name` indicates the `Repository` crd name that holds necessary backend information where backed up data will be stored.

#### spec.task

`spec.task` specifies the name and parameters of the `Task` template to use to restore the target data.

- **spec.task.name:** `spec.task.name` indicates the name of the [Task](/docs/concepts/crds/task.md) template to use for this restore process.
- **spec.task.params:** `spec.task.params` is an array of custom parameters to use to configure the task.

> `spec.task` section is not necessary for restoring workload data (i.e. Deployment, DaemonSet, StatefulSet etc.). However, it is necessary for restoring database and stand-alone PVC.

#### spec.target

`spec.target` field indicates the target where data will be restored. This section consist of following fields:

- **spec.target.ref :** `spec.target.ref` refers to the restore target. You have to specify `apiVersion`, `kind` and `name` of the target. Stash will use this information to inject an `init-container` or to create a restore job.

- **spec.target.volumeMounts :** `spec.target.volumeMounts` list of volumes and their `mountPath` where the data will be restored. Stash will mount these volumes inside the `init-container` or restore job.

#### spec.rules

`spec.rules` is an array of restore rule that specifies how Stash should restore in an individual host. For example, Stash runs restore process in all pod's of a StatefulSet. You can configure this `spec.rules` section to control which data will be restored into which pod.

Each restore rule has the following fields:

- **subjects :** `subjects` field contains a list of host name who are subject to this rule. If `subjects` field is kept empty, it indicates that this rule is applicable to all hosts. In the sample `RestoreSession` given above, the first rule is applicable to `host-3` and `host-4` only and the second rule is applicable to all hosts.
- **sourceHost :** `sourceHost` specifies the name of host whose backup data will be restored by this rule. In the sample `RestoreSession`, the first rule specify that backed up data of `host-0` (i.e. `pod-0` of old StatefulSet) will be restored into `host-3` and `host-4` (i.e. `pod-3` and `pod-4` of new StatefulSet). If you keep `sourceHost` field empty as the second rule of the sample, data from a similar backup host will be restored on the respective restore host. Which means, backup data of `host-0` will be restored into `host-0`, backup data of `host-1` will be restored into `host-1` and so on.
- **paths :** `paths` specifies a list of directories that will be restored into the hosts who are subject to this rule.
- **snapshots :** `snapshots` specifies the list of snapshots that will be restored into the hosts who are subject to this rule.

  >Note that, if you specify `snapshots` section, you don't have to specify `paths` section because each snapshot contains backup data of only one directory. Thus, if you specify both `paths` and `snapshots` and if a path does not exist in the specified snapshot then restore process will fail.

When the restore process runs in a host, it starts matching rules from the beginning. When a rule is matched, it immediately takes backup according to that rule and completes it's backup process. No further rules are checked. So, if your rule section has multiple matching rules for a host, only the first matching rule will be applied to restore for this host.

> If no rule match for a host, no data will be restored on that host.

#### spec.runtimeSettings

`spec.runtimeSettings` allows to configure runtime environment for restore `init-container` or job. You can specify runtime settings in both pod level and container level.

- **spec.runtimeSettings.container**

  `spec.runtimeSettings.container` is used to configure restore init-container/job in container level. You can configure the following container level parameters,

|       Field       |                                                                                                              Usage                                                                                                               |
| :---------------: | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|    `resources`    | Compute resources required by restore init-container or restore job. To know how to manage resources for containers, please visit [here](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/). |
|  `livenessProbe`  | Periodic probe of restore init-container/job's container liveness. Container will be restarted if the probe fails.                                                                                                               |
| `readinessProbe`  | Periodic probe of restore init-container/job's container readiness. Container will be removed from service endpoints if the probe fails.                                                                                         |
|    `lifecycle`    | Actions that the management system should take in response to container lifecycle events.                                                                                                                                        |
| `securityContext` | Security options that restore init-container/job's container should run with. For more details, please visit [here](https://kubernetes.io/docs/concepts/policy/security-context/).                                               |
|      `nice`       | Set CPU scheduling priority for the restore process. For more details about `nice`, please visit [here](https://www.askapache.com/optimize/optimize-nice-ionice/#nice).                                                          |
|     `ionice`      | Set I/O scheduling class and priority for the restore process. For more details about `ionice`, please visit [here](https://www.askapache.com/optimize/optimize-nice-ionice/#ionice).                                            |

- **spec.runtimeSettings.pod**

  `spec.runtimeSettings.pod` is used to configure restore job in pod level. You can configure following pod level parameters,

|             Field              |                                                                                                                  Usage                                                                                                                   |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `serviceAccountName`           | Name of the `ServiceAccount` to use for restore job. Stash init-container will use the same `ServiceAccount` as the target.                                                                                                              |
| `nodeSelector`                 | Selector which must be true for restore job pod to fit on a node.                                                                                                                                                                        |
| `automountServiceAccountToken` | Indicates whether a service account token should be automatically mounted into the restore job's pod.                                                                                                                                    |
| `nodeName`                     | NodeName is used to request to schedule restore job's pod onto a specific node.                                                                                                                                                          |
| `securityContext`              | Security options that restore job's pod should run with. For more details, please visit [here](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/).                                                              |
| `imagePullSecrets`             | A list of secret names in the same namespace that will be used to pull image from private docker registry. For more details, please visit [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/). |
| `affinity`                     | Affinity and anti-affinity to schedule restore job's pod in the desired node. For more details, please visit [here](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity).                      |
| `schedulerName`                | Name of the scheduler that should dispatch the restore job.                                                                                                                                                                              |
| `tolerations`                  | Taints and Tolerations to ensure that restore job's pod is not scheduled in inappropriate nodes. For more details about `toleration`, please visit [here](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/).      |
| `priorityClassName`            | Indicates the restore job pod's priority class. For more details, please visit [here](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                       |
| `priority`                     | Indicates the restore job pod's priority value.                                                                                                                                                                                          |
| `readinessGates`               | Specifies additional conditions to be evaluated for Pod readiness. For more details, please visit [here](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate).                                          |
| `runtimeClassName`             | RuntimeClass is used for selecting the container runtime configuration. For more details, please visit [here](https://kubernetes.io/docs/concepts/containers/runtime-class/)                                                             |
| `enableServiceLinks`           | EnableServiceLinks indicates whether information about services should be injected into pod's environment variables.                                                                                                                     |

#### spec.tempDir

Stash mount an `emtpyDir` for holding temporary files. It is also used for `caching` for faster restore performance. You can configure the `emptyDir` using `spec.tempDir` section. You can also disable `caching` using this field. Following fields are configurable in `spec.tempDir` section:

- **spec.tempDir.medium :** Specifies the type of storage medium should back this directory.
- **spec.tempDir.sizeLimit :** Maximum limit of storage for this volume.
- **spec.tempDir.disableCaching :** Disable caching while restoring. This may negatively impact restore performance.

### RestoreSession `Status` Section

`.status` section of `RestoreSession` shows progress, stats and overall phase of the restore process. Restore init-container or job add their respective stats in `.status` section after they complete their task. `.status` section consist of following fields:

#### status.phase

`status.phase` indicates the overall phase of the restore process for this RestoreSession. `status.phase` will be `Succeeded` only if the phase of all hosts are `Succeeded`. If any of the hosts fail to complete restore, `status.phase` will be `Failed`.

#### status.totalHosts

A `RestoreSession` may trigger restore of multiple hosts. For example, all the pod's of a `Deployment`, `ReplicaSet` and `ReplicationController` mounts same volume. In this case, Stash will restore data only in one pod. Thus, the total number of hosts for these workloads will be 1. On the other hand, pods of `StatefulSet` and `DaemonSet` may have different volume mounted into different replica. In this case, Stash will restore data in all individual pods. Thus, the total number of hosts for these workloads will be number of replicas for `StatefulSet` and number of running daemon pods for `DaemonSet`.

#### status.sessionDuration

`status.sessionDuration` indicates the total time taken to complete restore of all hosts. It is simply the sum of restore duration of all individual hosts.

#### status.stats

`status.stats` section is an array of restore statistics of individual hosts. Each host adds their statistics in this array after completing their restore process.

Individual host stats entry consist of following fields:

- **hostname :** `hostname` indicates the name of the host.
- **phase :** `phase` indicates the restore phase of this host.
- **duration :** `duration` indicates the total time taken to complete restore process for this host.
- **error :** `error` shows the reason for failure if the restore process fails for this host.

## Next Steps

- Learn how restore of workloads data works from [here](/docs/guides/workloads/restore.md).
- Learn how restore of databases works from [here](/docs/guides/databases/restore.md).
- Learn how restore stand-alone PVC works from [here](/docs/guides/volumes/restore.md).

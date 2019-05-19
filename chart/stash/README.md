# Stash
[Stash by AppsCode](https://github.com/stashed/stash) - Backup your Kubernetes Volumes
## TL;DR;

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm install appscode/stash --name stash-operator --namespace kube-system
```

## Introduction

This chart bootstraps a [Stash controller](https://github.com/stashed/stash) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.8+

## Installing the Chart
To install the chart with the release name `stash-operator`:
```console
$ helm install appscode/stash --name stash-operator
```
The command deploys Stash operator on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `stash-operator`:

```console
$ helm delete stash-operator
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the Stash chart and their default values.


|              Parameter               |                                                                                Description                                                                                 |                          Default                          |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `replicaCount`                       | Number of stash operator replicas to create (only 1 is supported)                                                                                                          | `1`                                                       |
| `operator.registry`                  | Docker registry used to pull operator image                                                                                                                                | `appscode`                                                |
| `operator.repository`                | operator container image                                                                                                                                                   | `stash`                                                   |
| `operator.tag`                       | operator container image tag                                                                                                                                               | `0.8.3`                                                   |
| `pushgateway.registry`               | Docker registry used to pull Prometheus pushgateway image                                                                                                                  | `prom`                                                    |
| `pushgateway.repository`             | Prometheus pushgateway container image                                                                                                                                     | `pushgateway`                                             |
| `pushgateway.tag`                    | Prometheus pushgateway container image tag                                                                                                                                 | `v0.5.2`                                                  |
| `cleaner.registry`                   | Docker registry used to pull Webhook cleaner image                                                                                                                         | `appscode`                                                |
| `cleaner.repository`                 | Webhook cleaner container image                                                                                                                                            | `kubectl`                                                 |
| `cleaner.tag`                        | Webhook cleaner container image tag                                                                                                                                        | `v1.11`                                                   |
| `imagePullPolicy`                    | container image pull policy                                                                                                                                                | `IfNotPresent`                                            |
| `criticalAddon`                      | If true, installs Stash operator as critical addon                                                                                                                         | `false`                                                   |
| `logLevel`                           | Log level for operator                                                                                                                                                     | `3`                                                       |
| `affinity`                           | Affinity rules for pod assignment                                                                                                                                          | `{}`                                                      |
| `annotations`                        | Annotations applied to operator pod(s)                                                                                                                                     | `{}`                                                      |
| `nodeSelector`                       | Node labels for pod assignment                                                                                                                                             | `{}`                                                      |
| `tolerations`                        | Tolerations used pod assignment                                                                                                                                            | `{}`                                                      |
| `serviceAccount.create`              | If `true`, create a new service account                                                                                                                                    | `true`                                                    |
| `serviceAccount.name`                | Service account to be used. If not set and `serviceAccount.create` is `true`, a name is generated using the fullname template                                              | ``                                                        |
| `apiserver.groupPriorityMinimum`     | The minimum priority the group should have.                                                                                                                                | 10000                                                     |
| `apiserver.versionPriority`          | The ordering of this API inside of the group.                                                                                                                              | 15                                                        |
| `apiserver.enableValidatingWebhook`  | Enable validating webhooks for Stash CRDs                                                                                                                                  | true                                                      |
| `apiserver.enableMutatingWebhook`    | Enable mutating webhooks for Kubernetes workloads                                                                                                                          | true                                                      |
| `apiserver.ca`                       | CA certificate used by main Kubernetes api server                                                                                                                          | `not-ca-cert`                                             |
| `apiserver.disableStatusSubresource` | If true, disables status sub resource for crds. Otherwise enables based on Kubernetes version | `false`            |
| `apiserver.bypassValidatingWebhookXray` | If true, bypasses validating webhook xray checks           | `false`               |
| `apiserver.useKubeapiserverFqdnForAks`  | If true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 | `true`             |
| `apiserver.healthcheck.enabled`      | Enable readiness and liveliness probes                                                                                                                                     | `true`                                                    |
| `enableAnalytics`                    | Send usage events to Google Analytics                                                                                                                                      | `true`                                                    |
| `monitoring.agent`                   | Specify which monitoring agent to use for monitoring Stash. It accepts either `prometheus.io/builtin` or `prometheus.io/coreos-operator`.                                  | `none`                                                    |
| `monitoring.backup`                  | Specify whether to monitor Stash backup and recovery.                                                                                                                      | `false`                                                   |
| `monitoring.operator`                | Specify whether to monitor Stash operator.                                                                                                                                 | `false`                                                   |
| `monitoring.prometheus.namespace`    | Specify the namespace where Prometheus server is running or will be deployed.                                                                                              | Release namespace                                         |
| `monitoring.serviceMonitor.labels`   | Specify the labels for ServiceMonitor. Prometheus crd will select ServiceMonitor using these labels. Only usable when monitoring agent is `prometheus.io/coreos-operator`. | `app: <generated app name>` and `release: <release name>` |
| `additionalPodSecurityPolicies`      | Additional psp names passed to operator                                                                                                                                    | `[]`                                                      |
| `platform.openshift`                 | Name of platform (eg: Openshift, AKS, EKS, GKE, etc.)                                                                                                                      | `false`                                                   |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
$ helm install --name stash-operator --set image.tag=v0.2.1 appscode/stash
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```console
$ helm install --name stash-operator --values values.yaml appscode/stash
```



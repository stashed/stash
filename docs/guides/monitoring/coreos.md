---
title: CoreOS Prometheus Operator | Stash
description: Monitor Stash using CoreOS Prometheus operator
menu:
  product_stash_0.7.0:
    identifier: monitoring-coreos-operator
    name: CoreOS Prometheus Operator
    parent: monitoring
    weight: 30
product_name: stash
menu_name: product_stash_0.7.0
---

# Monitoring Using CoreOS Prometheus Operator

CoreOS [prometheus-operator](https://github.com/coreos/prometheus-operator) provides simple and kubernetes native way to deploy and configure prometheus server. This tutorial will show you how to use CoreOS Prometheus operator for monitoring Stash.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

To keep Prometheus resources isolated, we will use a separate namespace to deploy Prometheus operator and respective resources.

```console
$ kubectl create ns demo
namespace/demo created
```

**Deploy Prometheus Operator:**

If you already don't have a CoreOS Prometheus operator running, create one using following command,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/prometheus-operator.yaml
clusterrole.rbac.authorization.k8s.io/prometheus-operator created
clusterrolebinding.rbac.authorization.k8s.io/prometheus-operator created
serviceaccount/prometheus-operator created
deployment.apps/prometheus-operator created
```

Now, wait for prometheus operator pod to be ready,

```console
$ kubectl get pods -n demo -l k8s-app=prometheus-operator
NAME                                   READY   STATUS    RESTARTS   AGE
prometheus-operator-6547d55767-vnlld   1/1     Running   0          1m
```

You can also follow the official docs to deploy prometheus operator from [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md).

## Enable Monitoring in Stash

Enable Prometheus monitoring using `coreos-operator` agent while installing Stash. To know details about how to enable monitoring see [here](/docs/guides/monitoring/overview.md#how-to-enable-monitoring).

Here, we are going to enable monitoring for both `backup & recovery` and `opeartor` metrics.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0/hack/deploy/stash.sh \
    | bash -s -- --monitoring-agent=coreos-operator --monitoring-backup=true --monitoring-operator=true --prometheus-namespace=demo
```

This will create a `ServiceMonitor` crd with name `stash-servicemonitor` in demo namespace for monitoring endpoints of `stash-operator` service.

Let's check the ServiceMonitor crd using following command,

```yaml
$ kubectl get servicemonitor stash-servicemonitor -n demo -o yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","metadata":{"annotations":{},"labels":{"app":"stash"},"name":"stash-servicemonitor","namespace":"demo"},"spec":{"endpoints":[{"port":"pushgateway"},{"bearerTokenFile":"/var/run/secrets/kubernetes.io/serviceaccount/token","port":"admission","scheme":"https","tlsConfig":{"caFile":"/etc/prometheus/secrets/stash-apiserver-cert/tls.crt","serverName":"stash-operator.kube-system.svc"}}],"namespaceSelector":{"matchNames":["kube-system"]},"selector":{"matchLabels":{"app":"stash"}}}}
  creationTimestamp: 2018-11-09T15:45:13Z
  generation: 1
  labels:
    app: stash
  name: stash-servicemonitor
  namespace: demo
  resourceVersion: "60655"
  selfLink: /apis/monitoring.coreos.com/v1/namespaces/demo/servicemonitors/stash-servicemonitor
  uid: 72a41740-e436-11e8-ab7d-0800277c8cd7
spec:
  endpoints:
  - port: pushgateway
  - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    port: admission
    scheme: https
    tlsConfig:
      caFile: /etc/prometheus/secrets/stash-apiserver-cert/tls.crt
      serverName: stash-operator.kube-system.svc
  namespaceSelector:
    matchNames:
    - kube-system
  selector:
    matchLabels:
      app: stash
```

Here, we have two endpoints at `spec.endpoints` field. One is `pushgateway` that exports backup and recovery metrics and another is `admission` which exports operator metrics.

Stash exports operator metrics in TLS secure `admission` endpoint. So, Prometheus server need to provide certificate while scrapping metrics from this endpoint. Stash has created a secret named `stash-apiserver-certs` with this certificate in `demo` namespace as we have specified that we will deploy Prometheus in that namespace through `--prometheus-namespace` flag. We have to specify this secret in Prometheus crd through `spec.secrets` field. Prometheus operator will mount this secret at `/etc/prometheus/secrets/stash-apiserver-cert` directory of respective Prometheus pod. So, we need to configure `tlsConfig` field to use that certificate. Here, `caFile` indicates the certificate to use and `serverName` is used to verify hostname. In our case, the certificate is valid for hostname `server` and `stash-operator.kube-system.svc`.

Let's check secret `stash-apiserver-cert` has been created in demo namespace.

```console
$ kubectl get secret -n demo -l=app=stash
NAME                   TYPE                DATA   AGE
stash-apiserver-cert   kubernetes.io/tls   2      31m
```
Also note that, there is a `bearerTokenFile` field. This file is token for the serviceaccount that will be created while creating RBAC stuff for Prometheus crd. This is required for authorizing Prometheus to Stash API Server.

Now, we are ready to deploy Prometheus server.

## Deploy Prometheus Server

In order to deploy Prometheus server, we have to create `Prometheus` crd. Prometheus crd defines a desired Prometheus server setup. For more details about `Prometheus` crd, please visit [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/design.md#prometheus).

**Create RBAC:**

If you are using a RBAC enabled cluster, you have to give necessary permissions to Prometheus. Let's create necessary RBAC stuffs for Prometheus.

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0//docs/examples/monitoring/coreos/prometheus-rbac.yaml 
clusterrole.rbac.authorization.k8s.io/prometheus created
serviceaccount/prometheus created
clusterrolebinding.rbac.authorization.k8s.io/prometheus created
```

**Create Prometheus:**

Below is the YAML of `Prometheus` crd that we are going to create for this tutorial,

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: demo
  labels:
    prometheus: prometheus
spec:
  replicas: 1
  serviceAccountName: prometheus
  serviceMonitorSelector:
    matchLabels:
      app: stash
  secrets:
  - stash-apiserver-cert
  resources:
    requests:
      memory: 400Mi
```

Here, `spec.serviceMonitorSelector` is used to select the `ServiceMonitor` crd that is created by Stash. We have provided `stash-apiserver-cert` secret in `spec.secrets` field. This will be mounted in Prometheus pod.

Let's create the `Prometheus` object we have shown above,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/coreos/prometheus.yaml
prometheus.monitoring.coreos.com/prometheus created
```

Prometheus operator watches for `Prometheus` crd. Once a `Prometheus` crd is created, Prometheus operator generates respective configuration and creates a StatefulSet to run Prometheus server.

Let's check StatefulSet has been created,

```console
$ kubectl get statefulset -n demo
NAME                    DESIRED   CURRENT   AGE
prometheus-prometheus   1         1         4m
```

Check StatefulSet's pod is running,

```console
$ kubectl get pod prometheus-prometheus-0 -n demo
NAME                      READY   STATUS    RESTARTS   AGE
prometheus-prometheus-0   2/2     Running   0          6m
```

Now, we are ready to access Prometheus dashboard.

### Verify Monitoring Metrics

Prometheus server is running on port `9090`. We will use [port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to access Prometheus dashboard. Run following command on a separate terminal,

```console
$ kubectl port-forward -n demo prometheus-prometheus-0 9090
Forwarding from 127.0.0.1:9090 -> 9090
Forwarding from [::1]:9090 -> 9090
```

Now, we can access the dashboard at `localhost:9090`. Open `localhost:9090` in your browser. You should see `pushgateway` and `addmision` endpoints of `stash-operator` service as target.

<p align="center">
  <img alt="Prometheus Target"  src="/docs/images/monitoring/prom-coreos-target.png", style="padding:10px">
</p>

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
# cleanup prometheus resources
kubectl delete -n demo prometheus prometheus
kubectl delete -n demo clusterrolebinding prometheus
kubectl delete -n demo clusterrole prometheus
kubectl delete -n demo serviceaccount prometheus
kubectl delete -n demo service prometheus-operated
kubectl delete -n demo secret stash-apiserver-cert

# cleanup prometheus operator resources
kubectl delete -n demo deployment prometheus-operator
kubectl delete -n dmeo serviceaccount prometheus-operator
kubectl delete clusterrolebinding prometheus-operator
kubectl delete clusterrole prometheus-operator

# delete namespace
kubectl delete ns demo
```

To uninstall Stash follow this [guide](/docs/setup/uninstall.md).

## Next Steps

- Learn how monitoring in Stash works from [here](/docs/guides/monitoring/overview.md).
- Learn how to monitor Stash using builtin Prometheus from [here](/docs/guides/monitoring/builtin.md).
- Learn how to use Grafana dashboard to visualize monitoring data from [here](/docs/guides/monitoring/grafana.md).
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
$ kubectl get pods -l k8s-app=prometheus-operator
NAME                                   READY   STATUS    RESTARTS   AGE
prometheus-operator-6547d55767-vnlld   1/1     Running   0          1m
```

You can also follow the official docs to deploy prometheus operator from [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md).

## Enable Monitoring in Stash

Enable Prometheus monitoring using `coreos-operator` agent while installing Stash. To know details about how to enable monitoring see [here](/docs/guides/monitoring/overview.md#how-to-enable-monitoring).

Here, we are going to enable monitoring for both `backup & recovery` and `opeartor` metrics.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.7.0/hack/deploy/stash.sh \
    | bash -s -- --monitoring-agent=coreos-operator --monitoring-backup=true --monitoring-operator=true
```

This will create a `ServiceMonitor` crd with name `stash-servicemonitor` in the same namespace as Stash operator for monitoring endpoints of `stash-operator` service.

Let's check the ServiceMonitor crd using following command,

```yaml
$ kubectl get servicemonitor stash-servicemonitor -n kube-system -o yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","metadata":{"annotations":{},"labels":{"app":"stash"},"name":"stash-servicemonitor","namespace":"kube-system"},"spec":{"endpoints":[{"port":"pushgateway"},{"bearerTokenFile":"/var/run/secrets/kubernetes.io/serviceaccount/token","port":"admission","scheme":"https","tlsConfig":{"caFile":"/etc/prometheus/secrets/stash-apiserver-cert/tls.crt","serverName":"stash-operator.kube-system.svc"}}],"namespaceSelector":{"any":true},"selector":{"matchLabels":{"app":"stash"}}}}
  creationTimestamp: 2018-11-07T08:47:39Z
  generation: 1
  labels:
    app: stash
  name: stash-servicemonitor
  namespace: kube-system
  resourceVersion: "21660"
  selfLink: /apis/monitoring.coreos.com/v1/namespaces/kube-system/servicemonitors/stash-servicemonitor
  uid: c82c0ec1-e269-11e8-a768-080027767ca3
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
    any: true
  selector:
    matchLabels:
      app: stash
```

Here, we have two endpoints at `spec.endpoints` field. One is `pushgateway` that exports backup and recovery metrics and another is `admission` which exports operator metrics.

Stash exports operator metrics in TLS secure `admission` endpoint. So, Prometheus server need to provide certificate while scrapping metrics from this endpoint. Stash creates a secret named `stash-apiserver-certs` with this certificate in the same namespace as Stash operator. We have to specify this secret in Prometheus crd through `spec.secrets` field. Prometheus operator will mount this secret at `/etc/prometheus/secrets/stash-apiserver-cert` directory of respective Prometheus pod. So, we need to configure `tlsConfig` field to use that certificate. Here, `caFile` indicates the certificate to use and `serverName` is used to verify hostname. In our case, the certificate is valid for hostname `server` and `stash-operator.kube-system.svc`.

Also note that, there is a `bearerTokenFile` field. This file is token for the serviceaccount that will be created while creating RBAC stuff for Prometheus crd. This is required for authorizing Prometheus to Stash API Server.

Now, we are ready to deploy Prometheus server.

## Deploy Prometheus Server

As we need TLS secure connection for `admission` endpoint, we have to mount the `stash-apiserver-cert` certificate to Prometheus pod. Stash has created this certificate in `kube-system` namespace. So, we have to deploy Prometheus in `kube-system` namespace.

>If you want to deploy Prometheus in a different namespace, you have to make a copy of this secret to that namespace.

**Create RBAC:**

If you are using a RBAC enabled cluster, you have to give necessary permissions to Prometheus. Let's create necessary RBAC stuffs for Prometheus.

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0//docs/examples/monitoring/coreos/prometheus-rbac.yaml 
clusterrole.rbac.authorization.k8s.io/prometheus created
serviceaccount/prometheus created
clusterrolebinding.rbac.authorization.k8s.io/prometheus created
```

**Create Prometheus:**

Now, we have to create `Prometheus` crd. Prometheus crd defines a desired Prometheus server setup. For more details about `Prometheus` crd, please visit [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/design.md#prometheus).

Below is the YAML of `Prometheus` crd that we are going to create for this tutorial,

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: kube-system
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
$ kubectl get statefulset -n kube-system
NAME                    DESIRED   CURRENT   AGE
prometheus-prometheus   1         1         4m
```

Check StatefulSet's pod is running,

```console
$ kubectl get pod prometheus-prometheus-0 -n kube-system
NAME                      READY   STATUS    RESTARTS   AGE
prometheus-prometheus-0   2/2     Running   0          6m
```

Now, we are ready to access Prometheus dashboard.

### Verify Monitoring Metrics

Prometheus server is running on port `9090`. We will use [port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to access Prometheus dashboard. Run following command on a separate terminal,

```console
$ kubectl port-forward -n kube-system prometheus-prometheus-0 9090
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
kubectl delete -n kube-system prometheus prometheus
kubectl delete -n kube-system clusterrolebinding prometheus
kubectl delete -n kube-system clusterrole prometheus
kubectl delete -n kube-system serviceaccount prometheus
kubectl delete -n kube-system service prometheus-operated

# cleanup prometheus operator resources
kubectl delete deployment prometheus-operator
kubectl delete serviceaccount prometheus-operator
kubectl delete clusterrolebinding prometheus-operator
kubectl delete clusterrole prometheus-operator
```

To uninstall Stash follow this [guide](/docs/setup/uninstall.md).

## Next Steps

- Learn how monitoring in Stash works from [here](/docs/guides/monitoring/overview.md).
- Learn how to monitor Stash using builtin Prometheus from [here](/docs/guides/monitoring/builtin.md).
- Learn how to use Grafana dashboard to visualize monitoring data from [here](/docs/guides/monitoring/grafana.md).
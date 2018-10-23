---
title: Monitoring | Stash
description: monitoring of Stash
menu:
  product_stash_0.7.0:
    identifier: monitoring-stash
    name: Monitoring
    parent: guides
    weight: 60
product_name: stash
menu_name: product_stash_0.7.0
section_menu_id: guides
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Monitoring Stash

Stash has native support for monitoring via Prometheus. This tutorial will show you how to monitor Stash with [CoreOS Prometheus Operator](https://github.com/coreos/prometheus-operator).

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

Now, install `Stash` in your cluster following the steps [here](/docs/setup/install.md).

To keep things isolated, we will use a separate namespace called `demo`. Create `demo` namespace if you already don't have one.

```console
$ kubectl create ns demo
namespace/demo created
```

We need some workloads with backup running through stash. If you already don't have any backup running, create one following the tutorial [here](/docs/guides/backup.md).

>Note: YAML files used in this tutorial can be found in [docs/examples/monitoring](/docs/examples/monitoring) folder in [appscode/stash](https://github.com/appscode/stash) github repository.

## Overview

Stash uses [Prometheus PushGateway](https://github.com/prometheus/pushgateway) to export the metrics. Following diagram shows the logical structure of stash monitoring flow.

<p align="center">
  <img alt="Monitoring Structure"  src="/docs/images/monitoring/stash-monitoring-structure.png">
</p>

Stash operator runs two containers. The `operator` container runs controller and other necessary stuff and the `pushgateway` container runs [prom/pushgateway](https://hub.docker.com/r/prom/pushgateway) image. Stash sidecar from different workloads pushes their metrics to this pushgateway. Then prometheus server scraps these metrics through `stash-operator` service.

### Exported Metrics

Currently, stash exports following metrics,

|                Metric                 |                      Uses                       |
| ------------------------------------- | ----------------------------------------------- |
| restic_session_success                | Indicates if session was successfully completed |
| restic_session_fail                   | Indicates if session failed                     |
| restic_session_duration_seconds_total | Total seconds taken to complete restic session  |
| restic_session_duration_seconds       | Total seconds taken to complete restic session  |

## Monitoring Using CoreOS Prometheus Operator

CoreOS [prometheus-operator](https://github.com/coreos/prometheus-operator) provides simple and kubernetes native way to deploy and configure prometheus server. This tutorial will show how to deploy prometheus operator and configure prometheus server for monitoring Stash. If you already have a prometheuse operator running, you can skip deploying prometheuse operator part.

### Deploy Prometheus Operator

In order to make deploying prometheus operator easier, we have put all necessary stuffs into a single yaml file. Deploy prometheus operator using following command,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/prometheus-operator.yaml
clusterrole.rbac.authorization.k8s.io/prometheus-operator created
clusterrolebinding.rbac.authorization.k8s.io/prometheus-operator created
serviceaccount/prometheus-operator created
deployment.extensions/prometheus-operator created
```

Now, wait for prometheus operator pod to be ready,

```console
$ kubectl get pods -n demo
NAME                                   READY   STATUS    RESTARTS   AGE
prometheus-operator-6547d55767-vnlld   1/1     Running   0          1m
```

You can also follow the official docs to deploy prometheus operator from [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md).

### Deploy Prometheus Server

Once the prometheus operator is ready, we are ready to deploy prometheus server. For a RBAC enabled cluster, we have to give necessary permissions for prometheus server to scrap the metrics.

#### Create RBAC Rules

Let's create necessary RBAC stuff first. If you don't have RBAC enabled in your cluster, you can skip this part.

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/prometheus-rbac.yaml 
clusterrole.rbac.authorization.k8s.io/prometheus created
clusterrolebinding.rbac.authorization.k8s.io/prometheus created
serviceaccount/prometheus created
```

#### Create ServiceMonitor

ServiceMonitor is used to select a set of services dynamically to be monitored through a prometheus server. For more details about `ServiceMonitor`, please visit [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/design.md#servicemonitor).

When installing Stash, you have also created a service with same name(i.e. stash-operator) as stash operator in same namespace. For our case, we have installed stash in `kube-system` namespace.

```console
$ kubectl get service -n kube-system -l=app=stash
NAME             TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)             AGE
stash-operator   ClusterIP   10.98.179.52   <none>        443/TCP,56789/TCP   3h
```

Let's view the yaml of `stash-operator` service.

```console
$ kubectl get service -n kube-system stash-operator -o yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Service","metadata":{"annotations":{},"labels":{"app":"stash"},"name":"stash-operator","namespace":"kube-system"},"spec":{"ports":[{"name":"admission","port":443,"targetPort":8443},{"name":"pushgateway","port":56789,"targetPort":56789}],"selector":{"app":"stash"}}}
  creationTimestamp: 2018-10-22T05:29:36Z
  labels:
    app: stash
  name: stash-operator
  namespace: kube-system
  resourceVersion: "2109"
  selfLink: /api/v1/namespaces/kube-system/services/stash-operator
  uid: 7698aab6-d5bb-11e8-bc53-080027bae9ba
spec:
  clusterIP: 10.98.179.52
  ports:
  - name: admission
    port: 443
    protocol: TCP
    targetPort: 8443
  - name: pushgateway
    port: 56789
    protocol: TCP
    targetPort: 56789
  selector:
    app: stash
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

Here, port `pushgateway` is used to export prometheus metrics.

Now, we have to create a `ServiceMonitor` that target this service and `pushgateway` endpoint. Below the YAML for `ServiceMonitor` that select this service,

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: stash-demo
  namespace: demo
  labels:
    app: stash
spec:
  namespaceSelector:
    any: true
  selector:
    matchLabels:
      app: stash
  endpoints:
  - port: pushgateway
```

Here, `spec.namespaceSelector:` fields value `any: true` denotes that it will select services from any namespace that matches the labels specified through `spec.selector.matchLabels`. `spec.endpoints` is used to specify the ports to use for scrapping metrics.

Let's create the `ServiceMonitor` we have shown above,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/service-monitor.yaml
servicemonitor.monitoring.coreos.com/stash-demo created
```

#### Create Prometheus

Now, we have to create `Prometheus` crd. Prometheus crd defines a desired Prometheus server setup. For more details about `Prometheus` crd, please visit [here](https://github.com/coreos/prometheus-operator/blob/master/Documentation/design.md#prometheus).

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
  resources:
    requests:
      memory: 400Mi
```

Here, `spec.serviceMonitorSelector` is used to select the `ServiceMonitor` crd we have created before.

Let's create the `Prometheus` object we have shown above,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/prometheus.yaml
prometheus.monitoring.coreos.com/prometheus created
```

Prometheus operator watches for `Prometheus` crd. Once a `Prometheus` crd is created, Prometheus operator generates respective configuration and creates a statefulset to deploy prometheus server.

Let's check statefulset has been created,

```console
$ kubectl get statefulset -n demo
NAME                    DESIRED   CURRENT   AGE
prometheus-prometheus   1         1         4m
```

Check statefulset pod is running,

```console
$ kubectl get pod prometheus-prometheus-0 -n demo
NAME                      READY   STATUS    RESTARTS   AGE
prometheus-prometheus-0   2/2     Running   0          6m
```

Now, we are ready to access prometheus dashboard.

### Verify Monitoring Metrics

Prometheus server is running on port `9090`. We will use [port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to access prometheus dashboard. Run following command on a separate terminal,

```console
$ kubectl port-forward -n demo prometheus-prometheus-0 9090
Forwarding from 127.0.0.1:9090 -> 9090
Forwarding from [::1]:9090 -> 9090
```

Now, we can access the dashboard at `localhost:9090`. Open `localhost:9090` in your browser. You should see `pushgateway` endpoint of `stash-operator` service as one of the target.

<p align="center">
  <img alt="Prometheus Target"  src="/docs/images/monitoring/prom-target.png", style="padding:10px">
</p>

## Use Grafana Dashboard

Grafana provides an elegant graphical user interface to visualize data. You can create beautiful dashboard easily with a meaningful representation of your prometheus metrics.

### Deploy Grafana

If you already do not have a grafana instance running, let's deploy one in Kubernetes. Below the YAML for deploying grafana using a Deployment.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: demo
  labels:
    app: grafana
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:5.3.1
```

Let's create the deployment shown above,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/grafana/grafana.yaml
deployment.apps/grafana created
```

Wait for grafana pod to goes in running state,

```console
$ kubectl get pod -n demo -l=app=grafana
NAME                       READY   STATUS    RESTARTS   AGE
grafana-56596dfd74-qk8qg   1/1     Running   0          2m
```

Grafana is running on port `3000`. We will forward this port to access grafana UI. Run following command on a separate terminal,

```console
$ kubectl port-forward -n demo grafana-56596dfd74-qk8qg 3000
Forwarding from 127.0.0.1:3000 -> 3000
Forwarding from [::1]:3000 -> 3000
```

Now, we can access grafana UI in `localhost:3000`. Use `username: admin` and `password:admin` to login to the UI.

### Add Prometheus Data Source

We have to add our prometheus server `prometheus-prometheus-0` as data source of grafana. We will use a `ClusterIP` service to connect prometheus server with grafana. Let's create a service to select prometheus server `prometheus-prometheus-0`,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/prometheus/prom-service.yaml
service/prometheus created
```

Below the YAML for the service we have created above,

```yaml
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: demo
spec:
  type: ClusterIP
  ports:
  - name: web
    port: 9090
    protocol: TCP
    targetPort: web
  selector:
    prometheus: prometheus
```

Now, follow these steps to add the prometheus server as data source of Grafana UI.

1. From Grafana UI, go to `Configuration` option from sidebar and click on `Data Sources`.

    <p align="center">
      <img alt="Grafana: Data Sources"  src="/docs/images/monitoring/grafana/grafana-data-source-1.png" style="padding: 10px;">
    </p>

2. Then, click on `Add data source`.

    <p align="center">
      <img alt="Grafana: Add data source"  src="/docs/images/monitoring/grafana/grafana-data-source-2.png" style="padding: 10px;">
    </p>

3. Now, configure `Name`, `Type` and `URL` fields as specified below and keep rest of the configuration to their default value then click `Save&Test` button.
    - *Name: Stash* (you can give any name)
    - *Type: Prometheus*
    - *URL: http://prometheus.demo.svc:9090*
      (url format: http://{prometheus service name}.{namespace}.svc:{port})

    <p align="center">
      <img alt="Grafana: Configure data source"  src="/docs/images/monitoring/grafana/grafana-data-source-3.png" style="padding: 10px;">
    </p>

Once you have added prometheus data source successfully, you are ready to create a dashboard to visualize the metrics.

### Import Stash Dashboard

Stash has a preconfigured dashboard created by [Alexander Trost](https://github.com/galexrt). You can import the dashboard using dashboard id `4198` or you can download json configuration of the dashboard from [here](https://grafana.com/dashboards/4198).

Follow these steps to import the preconfigured stash dashboard,

1. From Grafana UI, go to `Create` option from sidebar and click on `import`.

    <p align="center">
        <img alt="Grafana: Import dashboard"  src="/docs/images/monitoring/grafana/grafana-import-1.png" style="padding: 10px;">
    </p>

2. Then, insert the dashboard id `4198` in `Grafana.com Dashboard` field and press `Load` button. You can also upload `json` configuration file of the dashboard using `Upload .json File` button.

    <p align="center">
      <img alt="Grafana: Provide dashboard ID"  src="/docs/images/monitoring/grafana/grafana-import-2.png" style="padding: 10px;">
    </p>

3. Now on `prometheus-infra` field, select the data source name that we have given to our prometheus data source earlier. Then click on `Import` button.

    <p align="center">
        <img alt="Grafana: Select data source"  src="/docs/images/monitoring/grafana/grafana-import-3.png" style="padding: 10px;">
    </p>

Once you have imported the dashboard successfully, you will be greeted with Stash dashboard.

<p align="center">
      <img alt="Grafana: Stash dashboard"  src="/docs/images/monitoring/grafana/grafana-stash-dashboard.png" style="padding: 10px;">
</p>

## Monitoring Stash API Server

Stash operator runs [Extension API Server](https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/) to self-host webhooks and to provide `Snapshot` listing facility. This api-server exposes Prometheus native monitoring data on `/metrics` path via `admission` endpoint on `:8443` port.

You have to configure your prometheus server to scarp these metrics through `stash-operator` service.

We can configure our previous `ServiceMonitor` to scrap metrics from both `pushgateway` and `admission` endpoints as following,

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: stash-demo
  namespace: demo
  labels:
    app: stash
spec:
  namespaceSelector:
    any: true
  selector:
    matchLabels:
      app: stash
  endpoints:
  - port: pushgateway
  - port: admission
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    scheme: https
    tlsConfig:
      insecureSkipVerify: true
```

Here, we have added a new endpoints `admission`. API server runs TLS secure connection on port `8443`. Hence, we have specified `scheme: https` and `tlsConfig.insecureSkipVerify: true` to avoid TLS verification. We have also specified `bearerTokenFile` path of `prometheuse` serviceaccount for authentication purpose.

Now, you can update your previous `ServiceMonitor` with new one using `kubectl apply` command. Prometheus operator will automatically update configuration for prometheus server to match new `ServiceMonitor` specification. This automatic update may take some time.

Once configuration update is completed, you will see `admission` endpoints in prometheus target.

<p align="center">
  <img alt="Prometheus Target 2"  src="/docs/images/monitoring/prom-target-2.png", style="padding:10px">
</p>

### Exported Stash API Server Metrics

Stash api server exports following metrics,

- apiserver_audit_event_total
- apiserver_client_certificate_expiration_seconds_bucket
- apiserver_client_certificate_expiration_seconds_count
- apiserver_client_certificate_expiration_seconds_sum
- apiserver_current_inflight_requests
- apiserver_request_count
- apiserver_request_latencies_bucket
- apiserver_request_latencies_count
- apiserver_request_latencies_sum
- apiserver_request_latencies_summary
- apiserver_request_latencies_summary_count
- apiserver_request_latencies_summary_sum
- apiserver_storage_data_key_generation_failures_total
- apiserver_storage_data_key_generation_latencies_microseconds_bucket
- apiserver_storage_data_key_generation_latencies_microseconds_count
- apiserver_storage_data_key_generation_latencies_microseconds_sum
- apiserver_storage_envelope_transformation_cache_misses_total
- authenticated_user_requests
- etcd_helper_cache_entry_count
- etcd_helper_cache_hit_count
- etcd_helper_cache_miss_count
- etcd_request_cache_add_latencies_summary
- etcd_request_cache_add_latencies_summary_count
- etcd_request_cache_add_latencies_summary_sum
- etcd_request_cache_get_latencies_summary
- etcd_request_cache_get_latencies_summary_count
- etcd_request_cache_get_latencies_summary_sum

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
# cleanup prometheus resources
kubectl delete -n demo servicemonitor stash-demo
kubectl delete -n demo prometheus prometheus
kubectl delete -n demo service prometheus
kubectl delete -n demo clusterrolebinding prometheus
kubectl delete -n demo clusterrole prometheus
kubectl delete -n demo serviceaccount prometheus

# cleanup prometheus operator resources
kubectl delete -n demo deployment prometheus-operator
kubectl delete -n demo serviceaccount prometheus-operator
kubectl delete -n demo clusterrolebinding prometheus-operator
kubectl delete -n demo clusterrole prometheus-operator
kubectl delete -n demo service prometheus-operated

# cleanup grafana resources
kubectl delete -n demo deployment grafana

# cleanup namespace
kubectl delete namespace demo
```

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/guides/backup.md).
- Learn about the details of Restic CRD [here](/docs/concepts/crds/restic.md).
- To restore a backup see [here](/docs/guides/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concepts/crds/recovery.md).
- To run backup in offline mode see [here](/docs/guides/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/guides/backends.md).
- See working examples for supported workload types [here](/docs/guides/workloads.md).
- Learn about how to configure [RBAC roles](/docs/guides/rbac.md).
- Want to hack on Stash? Check our [contribution guidelines](/docs/CONTRIBUTING.md).

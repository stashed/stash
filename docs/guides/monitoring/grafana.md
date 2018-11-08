---
title: Use Grafana | Stash
description: Using Grafana dashboard to visualize Stash monitoring data
menu:
  product_stash_0.7.0:
    identifier: monitoring-grafana
    name: Use Grafana
    parent: monitoring
    weight: 40
product_name: stash
menu_name: product_stash_0.7.0
---

# Use Grafana Dashboard

Grafana provides an elegant graphical user interface to visualize data. You can create beautiful dashboard easily with a meaningful representation of your Prometheus metrics.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

You must have a Stash instant running with monitoring enabled. You can enable monitoring by following the guides for [Builtin Prometheus](/docs/guides/monitoring/builtin.md) or [CoreOS Prometheus Operator](/docs/guides/monitoring/coreos.md). For this tutorial, we have enabled Prometheus monitoring using CoreOS Prometheus operator.

## Deploy Grafana

If you already do not have a grafana instance running, let's deploy one in Kubernetes. Below the YAML for deploying grafana using a Deployment.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
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

Let's create the deployment we have shown above,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0/docs/examples/monitoring/grafana/grafana.yaml
deployment.apps/grafana created
```

Wait for grafana pod to goes in running state,

```console
$ kubectl get pod -l=app=grafana
NAME                       READY   STATUS    RESTARTS   AGE
grafana-7f594dc9c6-xwkf2   1/1     Running   0          3m22s
```

Grafana is running on port `3000`. We will forward this port to access grafana UI. Run following command on a separate terminal,

```console
$kubectl port-forward grafana-7f594dc9c6-xwkf2 3000
Forwarding from 127.0.0.1:3000 -> 3000
Forwarding from [::1]:3000 -> 3000
```

Now, we can access grafana UI in `localhost:3000`. Use `username: admin` and `password:admin` to login to the UI.

### Add Prometheus Data Source

We have to add our prometheus server `prometheus-prometheus-0` as data source of grafana. We will use a `ClusterIP` service to connect prometheus server with grafana. Let's create a service to select prometheus server `prometheus-prometheus-0`,

```console
$ kubectl apply -f https://raw.githubusercontent.com/appscode/stash/0.7.0//docs/examples/monitoring/coreos/prometheus-service.yaml
service/prometheus created
```

Below the YAML for the service we have created above,

```yaml
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: kube-system
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

Now, follow these steps to add the Prometheus server as data source of Grafana UI.

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
    - *URL: http://prometheus.kube-system.svc:9090*
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

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
kubectl delete service -n kube-system prometheus
kubectl delete deployment grafana
```

## Next Steps

- Learn how monitoring in Stash works from [here](/docs/guides/monitoring/overview.md).
- Learn how to monitor Stash using builtin Prometheus from [here](/docs/guides/monitoring/builtin.md).
- Learn how to monitor Stash using CoreOS Prometheus Operator from [here](/docs/guides/monitoring/coreos.md).

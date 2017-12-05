---
title: Monitoring | Stash
description: monitoring of Stash
menu:
  product_stash_0.5.1:
    identifier: monitoring-stash
    name: Monitoring
    parent: getting-started
    weight: 40
product_name: stash
menu_name: product_stash_0.5.1
section_menu_id: getting-started
url: /products/stash/0.5.1/getting-started/monitoring/
aliases:
  - /products/stash/0.5.1/monitoring/
---

> New to Stash? Please start [here](/docs/tutorials/README.md).

# Monitoring Stash

Stash has native support for monitoring via Prometheus.

## Monitoring Stash Operator
Stash operator exposes Prometheus native monitoring data via `/metrics` endpoint on `:56790` port. You can setup a [CoreOS Prometheus ServiceMonitor](https://github.com/coreos/prometheus-operator) using `stash-operator` service.

## Monitoring Backup Operation
Since backup operations are run as cron jobs, Stash can use [Prometheus Pushgateway](https://github.com/prometheus/pushgateway) cache metrics for backup operation. The installation scripts for Stash operator deploys a Prometheus Pushgateway as a sidecar container. You can configure a Prometheus server to scrape this Pushgateway via `stash-operator` service on port `:56789`. Backup operations send the following metrics to this Pushgateway:

 - `restic_session_success{job="<restic.namespace>-<restic.name>", app="<workload>"}`: Indicates if session was successfully completed
 - `restic_session_fail{job="<restic.namespace>-<restic.name>", app="<workload>"}`: Indicates if session failed
 - `restic_session_duration_seconds_total{job="<restic.namespace>-<restic.name>", app="<workload>"}`: Total seconds taken to complete restic session
 - `restic_session_duration_seconds{job="<restic.namespace>-<restic.name>", app="<workload>", filegroup="dir1", op="backup|forget"}`: Total seconds taken to complete restic session

## Next Steps

- Learn how to use Stash to backup a Kubernetes deployment [here](/docs/tutorials/backup.md).
- Learn about the details of Restic CRD [here](/docs/concept_restic.md).
- To restore a backup see [here](/docs/tutorials/restore.md).
- Learn about the details of Recovery CRD [here](/docs/concept_recovery.md).
- To run backup in offline mode see [here](/docs/tutorials/offline_backup.md)
- See the list of supported backends and how to configure them [here](/docs/tutorials/backends.md).
- See working examples for supported workload types [here](/docs/tutorials/workloads.md).
- Learn about how to configure [RBAC roles](/docs/tutorials/rbac.md).
- Learn about how to configure Stash operator as workload initializer [here](/docs/tutorials/initializer.md).
- Wondering what features are coming next? Please visit [here](/ROADMAP.md). 
- Want to hack on Stash? Check our [contribution guidelines](/CONTRIBUTING.md).
---
title: Stash Overview
description: Stash Overview
menu:
  product_stash_0.8.3:
    identifier: overview-concepts
    name: Overview
    parent: what-is-stash
    weight: 10
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---

# Stash

 Stash by AppsCode is a Kubernetes operator for [restic](https://restic.net). If you are running production workloads in Kubernetes, you might want to take backup of your disks, databases etc. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. `restic` is a backup program that is fast, efficient and secure with few moving parts. Stash is a CRD controller for Kubernetes built around `restic` to address these issues. Using Stash, you can backup Kubernetes volumes mounted in workloads, stand-alone volumes and databases.

## Features

|                                    Features                                     | Availability |                                                                         Scope                                                                         |
| ------------------------------------------------------------------------------- | :----------: | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Backup & Restore Workload Data                                                  |   &#10003;   | Deployment, DaemonSet, StatefulSet, ReplicaSet, ReplicationController, OpenShift DeploymentConfig                                                     |
| Backup & Restore Stand-alone Volume (PVC)                                       |   &#10003;   | PersistentVolumeClaim, PersistentVolume                                                                                                               |
| Backup & Restore Database                                                       |   &#10003;   | PostgreSQL, MySQL, MongoDB, ElasticSearch                                                                                                             |
| [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/) |   &#10003;   | CSI Driver must support VolumeSnapshot and Kubernetes Alpha features must be enabled                                                                  |
| Backup Kubernetes Cluster                                                       |   &#10003;   | Manual restore                                                                                                                                        |
| Schedule Backup                                                                 |   &#10003;   | Schedule through [cron expression](https://en.wikipedia.org/wiki/Cron)                                                                                |
| Instant Backup                                                                  |   &#10003;   | Use CLI or create BackupSession manually                                                                                                              |
| Default Backup                                                                  |   &#10003;   | Using a Template and annotations                                                                                                                      |
| Pause Scheduled Backup                                                          |   &#10003;   |                                                                                                                                                       |
| Support Multiple Storage Provider                                               |   &#10003;   | AWS S3, Minio, Rook, GCS, Azure, OpenStack Swift,  Backblaze B2, Rest Server, any PV/PVC                                                              |
| Encryption                                                                      |   &#10003;   | AES-256 in counter mode (CTR) (for Restic driver)                                                                                                     |
| Deduplication (send only diff)                                                  |   &#10003;   | Uses [Content Defined Chunking (CDC)](https://restic.net/blog/2015-09-12/restic-foundation1-cdc) (for Restic driver)                                  |
| Cleanup old snapshots automatically                                             |   &#10003;   | Cleanup according to different [retention policies](https://restic.readthedocs.io/en/stable/060_forget.html#removing-snapshots-according-to-a-policy) |
| Prometheus Metrics for Backup & Restore Process                                 |   &#10003;   | Official Prometheus Server, CoreOS Prometheus Operator                                                                                                |
| Prometheus Metrics for Stash Operator                                           |   &#10003;   | Official Prometheus Server, CoreOS Prometheus Operator                                                                                                |
| Support RBAC enabled cluster                                                    |   &#10003;   |                                                                                                                                                       |
| Support PSP enabled cluster                                                     |   &#10003;   |                                                                                                                                                       |
| CLI                                                                             |   &#10003;   | `kubectl` plugin (for Kubernetes 1.12+)                                                                                                               |
| Extensibility                                                                   |   &#10003;   | Extend using `Function` and `Task`                                                                                                                    |
| Customizability                                                                 |   &#10003;   | Customize backup / restore process using `Function` and `Task`                                                                                        |

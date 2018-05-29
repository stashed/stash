---
title: Stash Overview
description: Stash Overview
menu:
  product_stash_0.7.0:
    identifier: overview-concepts
    name: Overview
    parent: what-is-stash
    weight: 10
product_name: stash
menu_name: product_stash_0.7.0
section_menu_id: concepts
---
# Stash

 Stash by AppsCode is a Kubernetes operator for [restic](https://restic.net). If you are running production workloads in Kubernetes, you might want to take backup of your disks. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. `restic` is a backup program that is fast, efficient and secure with few moving parts. Stash is a CRD controller for Kubernetes built around `restic` to address these issues. Using Stash, you can backup Kubernetes volumes mounted in following types of workloads:

- Deployment
- DaemonSet
- ReplicaSet
- ReplicationController
- StatefulSet

## Features
 - Fast, secure, efficient backup of any kubernetes [volumes](https://kubernetes.io/docs/concepts/storage/volumes/).
 - Automates configuration of `restic` for periodic backup.
 - Store backed up files in various cloud storage provider, including S3, GCS, Azure, OpenStack Swift, DigitalOcean Spaces etc.
 - Restore backup easily.
 - Periodically check integrity of backed up data.
 - Take backup in offline mode.
 - Support workload initializer for faster backup.
 - Prometheus ready metrics for backup process.

---
title: Table of Contents | Guides
description: Table of Contents | Guides
menu:
  product_stash_0.8.3:
    identifier: latest-guides-readme
    name: Readme
    parent: latest-guides
    weight: -1
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
url: /products/stash/0.8.3/guides/latest/
aliases:
  - /products/stash/0.8.3/guides/latest/README/
---

# Guides

Guides show how to perform different operations with Stash. We have divided guides section into the following sub-sections:

- **Supported Backends :** This section describes how to configure different backend for Stash. It shows how to create `Storage Secret` and `Repository` object for different backends. Start learning to configure different backends from [here](/docs/guides/latest/backends/overview.md).

- **Workloads :** Workloads section describes how to backup and restore data from/to inside various Kubernetes resources such as Deployment, StatefulSet, DaemonSet, ReplicaSet etc. Get started with backup workloads data using Stash from [here](/docs/guides/latest/workloads/backup.md).

- **Volumes :** Volumes section describes how to backup and restore stand-alone Kubernetes volumes using Stash. Learn how to backup Kubernetes volumes using Stash from [here](/docs/guides/latest/volumes/backup.md).

- **Databases :** Databases section describes how to backup and restore databases using Stash. Check how database backup works in Stash from [here](/docs/guides/latest/databases/backup.md).

- **Default Backup :** This section describes how you can write template for backup process to backup resources using some annotations. Start learning to write backup template from [here](/docs/guides/latest/default-backup/overview.md).

- **Advanced Use Cases :** This section describes some advance uses of Stash such as instant backup, restoring in different namespace/cluster, restoring in different host etc. See how to trigger an instant backup from [here](/docs/guides/latest/advanced/instant_backup.md).

- **Monitoring :** Monitoring section describes how to monitor backup and restore process using Prometheus.

- **Stash CLI :** Stash provides a CLI to perform various operations such as unlocking a repository, restoring backed up data locally, triggering backup instantly etc. This section describes how you can perform such operations.

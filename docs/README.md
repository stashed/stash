---
title: Overview | Stash
description: Overview of Stash
menu:
  product_stash_0.5.1:
    identifier: overview-stash
    name: Overview
    parent: getting-started
    weight: 20
product_name: stash
menu_name: product_stash_0.5.1
section_menu_id: getting-started
url: /products/stash/0.5.1/getting-started/
aliases:
  - /products/stash/0.5.1/
  - /products/stash/0.5.1/README/
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

From here you can learn all about Stash's architecture and how to deploy and use Stash.

- [Concepts](/docs/concepts/). Concepts explain some significant aspect of Stash. This is where you can learn about what Stash does and how it does it.

- [Setup](/docs/setup/). Setup contains instructions for installing
  the Stash in various cloud providers.

- [Guides](/docs/guides/). Guides show you how to perform tasks with Stash.

- [Reference](/docs/reference/). Detailed exhaustive lists of
command-line options, configuration options, API definitions, and procedures.

We're always looking for help improving our documentation, so please don't hesitate to [file an issue](https://github.com/appscode/stash/issues/new) if you see some problem. Or better yet, submit your own [contributions](/docs/CONTRIBUTING.md) to help
make our docs better.

---

**The stash operator collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--analytics=false`.

---

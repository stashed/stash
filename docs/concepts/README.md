---
title: Concepts | Stash
menu:
  product_stash_0.8.3:
    identifier: concepts-readme
    name: README
    parent: concepts
    weight: -1
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
url: /products/stash/0.8.3/concepts/
aliases:
  - /products/stash/0.8.3/concepts/README/
---

# Concepts

Concepts help you to learn about the different parts of the Stash and the abstractions it uses.

This concept section is divided into the following modules:

- What is Stash?
  - [Overview](/docs/concepts/what-is-stash/overview.md) provides an introduction to Stash. It also give an overview of the features it provides.
  - [Architecture](/docs/concepts/what-is-stash/architecture.md) provides a visual representation of Stash architecture. It also provides a brief overview of the components it uses.

- Declarative API
  - [Repository](/docs/concepts/crds/repository.md) introduces concept of `Repository` crd that holds backend information in a Kubernetes native way.
  - [BackupConfiguration](/docs/concepts/crds/backupconfiguration.md) introduces concept of `BackupConfiguration` crd that is used to configure backup for a target resource in Kubernetes native way.
  - [BackupSession](/docs/concepts/crds/backupsession.md) introduces concept of `BackupSession` crd that represents a backup instance of a target resource for respective `BackupConfiguration` object.
  - [RestoreSession](/docs/concepts/crds/restoresession.md) introduces concept of `RestoreSession` crd that represents a restore instance of a target resource.
  - [Function](/docs/concepts/crds/function.md) introduces concept of `Function` crd that represents a step of a backup or restore process.
  - [Task](/docs/concepts/crds/task.md) introduces concept of `Task` crd which specifies an ordered collection of multiple `Function` and their parameters that make up a complete backup or restore process.
  - [BackupConfigurationTemplate](/docs/concepts/crds/backupconfiguration_template.md) introduces concept of `BackupConfigurationTemplate` crd that specifies a template for `Repository` and `BackupConfiguration` object which provides option to backup using few annotations only.
  - [AppBinding](/docs/concepts/appbinding.md) introduces concept of `AppBinding` crd which holds the information that are necessary to connect with a database.
  - [Snapshot](/docs/concepts/crds/snapshot.md) introduces concept of `Snapshot` object that represents backed up snapshots in a Kubernetes native way.

- Old API
  - [Restic](/docs/concepts/old-crds/restic.md) introduces the concept of `Restic` crd that is used for configuring [restic](https://restic.net) in a Kubernetes native way.
  - [Recovery](/docs/concepts/old-crds/recovery.md) introduces the concept of `Recovery` crd that is used to restore a backup taken using Stash.

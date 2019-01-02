---
title: Concepts | Stash
menu:
  product_stash_0.8.2:
    identifier: concepts-readme
    name: Readme
    parent: concepts
    weight: -1
product_name: stash
menu_name: product_stash_0.8.2
section_menu_id: concepts
url: /products/stash/0.8.2/concepts/
aliases:
  - /products/stash/0.8.2/concepts/README/
---
# Concepts

Concepts help you learn about the different parts of the Stash and the abstractions it uses.

- What is Stash?
  - [Overview](/docs/concepts/what-is-stash/overview.md). Provides a conceptual introduction to Stash, including the problems it solves and its high-level architecture.
- Custom Resource Definitions
  - [Restic](/docs/concepts/crds/restic.md). Introduces the concept of `Restic` for configuring [restic](https://restic.net) in a Kubernetes native way.
  - [Recovery](/docs/concepts/crds/recovery.md). Introduces the concept of `Recovery` to restore a backup taken using Stash.
  - [Repository](/docs/concepts/crds/repository.md) Introduce concept of `Repository` that represents restic repository in a Kubernetes native way.
  - [Snapshot](/docs/concepts/crds/snapshot.md) Introduce concept of `Snapshot` that represents backed up snapshots in a Kubernetes native way.
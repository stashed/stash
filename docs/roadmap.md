---
title: Roadmap | Stash
description: Roadmap of stash
menu:
  product_stash_0.7.0:
    identifier: roadmap-stash
    name: Roadmap
    parent: welcome
    weight: 15
product_name: stash
menu_name: product_stash_0.7.0
section_menu_id: welcome
url: /products/stash/0.7.0/welcome/roadmap/
aliases:
  - /products/stash/0.7.0/roadmap/
---

# Project Status

## Versioning Policy
There are 2 parts to versioning policy:

 - Operator version: Stash __does not follow semver__. Currently Stash operator implementation is considered alpha. Please report any issues you via Github. Once released, the _major_ version of operator is going to point to the Kubernetes [client-go](https://github.com/kubernetes/client-go#branches-and-tags) version. You can verify this from the `glide.yaml` file. This means there might be breaking changes between point releases of the operator. This generally manifests as changed annotation keys or their meaning.
Please always check the release notes for upgrade instructions.
 - CRD version: `stash.appscode.com/v1alpha1` is considered in alpha. This means breaking changes to the YAML format
might happen among different releases of the operator.

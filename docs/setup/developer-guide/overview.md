---
title: Overview | Developer Guide
description: Developer Guide Overview
menu:
  product_stash_0.7.0-rc.1:
    identifier: developer-guide-readme
    name: Overview
    parent: developer-guide
    weight: 15
product_name: stash
menu_name: product_stash_0.7.0-rc.1
section_menu_id: setup
---

> New to Stash? Please start [here](/docs/concepts/README.md).

## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Stash.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Stash should not change.

### Build Stash
Some of the Stash development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

#### Setup GO
Stash is written in Google's GO programming language. Currently, Stash is developed and tested on **go 1.9.2**. If you haven't set up a GO
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install GO.

#### Download Source

```console
$ go get github.com/appscode/stash
$ cd $(go env GOPATH)/src/github.com/appscode/stash
```

#### Install Dev tools
To install various dev tools for Stash, run the following command:
```console
$ ./hack/builddeps.sh
```

#### Build Binary
```
$ ./hack/make.py
$ stash version
```

#### Run Binary Locally
```console
$ stash run \
  --secure-port=8443 \
  --kubeconfig="$HOME/.kube/config" \
  --authorization-kubeconfig="$HOME/.kube/config" \
  --authentication-kubeconfig="$HOME/.kube/config" \
  --authentication-skip-lookup
```

#### Dependency management
Stash uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. Dependencies are already checked in the `vendor` folder.
If you want to update/add dependencies, run:
```console
$ glide slow
```

#### Build Docker images
To build and push your custom Docker image, follow the steps below. To release a new version of Stash, please follow the [release guide](/docs/setup/developer-guide/release.md).

```console
# Build Docker image
$ ./hack/docker/setup.sh; ./hack/docker/setup.sh push

# Add docker tag for your repository
$ docker tag appscode/stash:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>
```

#### Generate CLI Reference Docs
```console
$ ./hack/gendocs/make.sh
```

### Testing Stash
#### Unit tests
```console
$ ./hack/make.py test unit
```

#### Run e2e tests
Stash uses [Ginkgo](http://onsi.github.io/ginkgo/) to run e2e tests.
```console
$ ./hack/make.py test e2e
```

To run e2e tests against remote backends, you need to set cloud provider credentials in `./hack/config/.env`. You can see an example file in `./hack/config/.env.example`.

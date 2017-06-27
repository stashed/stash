## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Stash.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Stash should not change.

### Build Stash
Some of the Stash development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

#### Setup GO
Stash is written in Google's GO programming language. Currently, Stash is developed and tested on **go 1.8.3**. If you haven't set up a GO
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install GO.

#### Download Source

```sh
$ go get github.com/appscode/stash
$ cd $(go env GOPATH)/src/github.com/appscode/stash
```

#### Install Dev tools
To install various dev tools for Stash, run the following command:
```sh
$ ./hack/builddeps.sh
```

#### Build Binary
```
$ ./hack/make.py
$ stash version
```

#### Dependency management
Stash uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. Dependencies are already checked in the `vendor` folder.
If you want to update/add dependencies, run:
```sh
$ glide slow
```

#### Build Docker images
To build and push your custom Docker image, follow the steps below. To release a new version of Stash, please follow the [release guide](/docs/developer-guide/release.md).

```sh
# Build Docker image
$ ./hack/docker/stash/setup.sh

# Add docker tag for your repository
$ docker tag appscode/stash:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>
```

#### Generate CLI Reference Docs
```sh
$ ./hack/gendocs/make.sh 
```

### Run Tests
#### Unit tests
```sh
go test -v ./pkg/...
```

#### Run e2e tests
Stash uses [Ginkgo](http://onsi.github.io/ginkgo/) to run e2e tests.

#### Run e2e Test
```
$ ./hack/make.py test minikube # Run Test against minikube, this requires minikube to be set up and started.

$ ./hack/make.py test e2e -cloud-provider=gce # Test e2e against gce cluster

$ ./hack/make.py test integration -cloud-provider=gce # Run Integration test against gce
                                                      # This requires stash to be deployed in the cluster.

```

```
- Run only one e2e test
$ ./hack/make.py test e2e -cloud-provider=gce -test-only=CoreIngress


- Run One test but do not delete all resource that are created
$ ./hack/make.py test minikube -cloud-provider=gce -test-only=CoreIngress -cleanup=false


- Run Service IP Persist test with provided IP
$ ./hack/make.py test e2e -cloud-provider=gce -test-only=CreateIPPersist -lb-ip=35.184.104.215

```

Tests are run only in namespaces prefixed with `test-`. So, to run tests in your desired namespace, follow these steps:
```
# create a Kubernetes namespace in minikube with
kubectl create ns test-<any-name-you-want>

# run tests
./hack/make.py test minikube -namespace test-<any-name-you-want> -max-test=1
```

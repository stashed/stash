## Development Guide

### Go development environment
Stash is written in the go programming language. The release is built and tested on **go 1.8**. If you haven't set up a Go
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install the go tools.

### Dependency management
Stash build and test scripts use glide to manage dependencies.

To install glide follow [these instructions](https://github.com/Masterminds/glide#install).

Currently the project includes all its required dependencies inside `vendor` to make things easier.

### Run Test
#### Run Unit Test by
```sh
./hack/make.py test unit
```

#### Run e2e Test
```sh
./hack/make.py test e2e
```

### Local Build
To build Stash using your local Go development environment (generate linux binaries):
```sh
$ ./hack/make.py
```
Read full [Build instructions](build.md).
[![PkgGoDev](https://pkg.go.dev/badge/gomodules.xyz/sets)](https://pkg.go.dev/gomodules.xyz/sets)

# sets

This library has been forked from https://github.com/kubernetes/apimachinery/tree/v0.18.3/pkg/util/sets so that it can easily used in non-k8s related projects.

## Generate set types

```bash
set-gen \
  --input-dirs=gomodules.xyz/sets/types \
  --output-package=gomodules.xyz/sets
```

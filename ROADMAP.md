# Project Status

## Versioning Policy
There are 2 parts to versioning policy:
 - Operator version: Stash __does not follow semver__. Currently Stash operator implementation is considered alpha. Please report any issues you via Github. Once released, the _major_ version of operator is going to point to the Kubernetes [client-go](https://github.com/kubernetes/client-go#branches-and-tags) version. You can verify this from the `glide.yaml` file. This means there might be breaking changes between point releases of the operator. This generally manifests as changed annotation keys or their meaning.
Please always check the release notes for upgrade instructions.
 - TPR version: `stash.appscode.com/v1alpha1` is considered in alpha. This means breaking changes to the YAML format
might happen among different releases of the operator.

### Release 0.x.0
These are pre-releases done so that users can test Stash.

### Release 3.0.0
This is going to be the first release of Stash and uses Kubernetes client-go 3.0.0. We plan to mark the last 0.x.0 release as 3.0.0. This version will support Kubernetes 1.5 & 1.6 .

### Release 4.0.0
This relased will be based on client-go 4.0.0. This is going to include a number of breaking changes (example, use CustomResoureDefinition instead of TPRs) and be supported for Kubernetes 1.7+. Please see the issues in release milestone [here](https://github.com/appscode/stash/milestone/3).

## Backend Support
Stash currently includes a forked version of [restic](https://github.com/restic/restic). You can find our forked version [here](https://github.com/appscode/restic). Our forked version adds support for [GCS](https://github.com/restic/restic/pull/1052) and [Azure](https://github.com/restic/restic/pull/1059) backend. If you would like these changes merged into upstream project, please upvote the corresponding issues.

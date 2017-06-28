[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/stash)](https://goreportcard.com/report/github.com/appscode/stash)

# Stash
 Stash by AppsCode is a Kubernetes operator for [restic](https://github.com/restic/restic). Using Stash, you can backup Kubernetes volumes mounted in following types of workloads:
- Deployment
- Stateful Set
- Replica Set
- Daemon Set
- Replication Controller

## Features
 - Fast, secure, efficient backup of any kubernetes [volumes](https://kubernetes.io/docs/concepts/storage/volumes/).
 - Automates configuration of `restic` for periodic backup.
 - Store backed up files in various cloud storage provider, including S3, GCS, Azure, etc.
 - Prometheus ready metrics for backup process.

## Supported Versions
Kubernetes 1.5+

## Installation
To install Stash, please follow the guide [here](/docs/install.md).

## Using Stash
Want to learn how to use Stash? Please start [here](/docs/tutorial.md).

## Contribution guidelines
Want to help improve Stash? Please start [here](/CONTRIBUTING.md).

## Versioning Policy
There are 2 parts to versioning policy:
 - Operator version: Stash __does not follow semver__, rather the _major_ version of operator points to the
Kubernetes [client-go](https://github.com/kubernetes/client-go#branches-and-tags) version.
You can verify this from the `glide.yaml` file. This means there might be breaking changes
between point releases of the operator. This generally manifests as changed annotation keys or their meaning.
Please always check the release notes for upgrade instructions.
 - TPR version: `stash.appscode.com/v1alpha1` is considered in alpha. This means breaking changes to the YAML format
might happen among different releases of the operator.

---

**The stash operator collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--analytics=false`.

---

## Acknowledgement
 - Many thanks for [Alexander Neumann](https://github.com/fd0) for [Restic](https://github.com/restic/restic) project.

## Support
If you have any questions, you can reach out to us.
* [Slack](https://slack.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
* [Website](https://appscode.com)

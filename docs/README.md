[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/stash)](https://goreportcard.com/report/github.com/appscode/stash)

# Stash
 Stash by AppsCode is a Kubernetes operator for [restic](https://github.com/restic/restic). If you are running production workloads in Kubernetes, you might want to take backup of your disks. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. `restic` is a backup program that is fast, efficient and secure with few moving parts. Stash is a TPR controller for Kubernetes built around `restic` to address these issues. Using Stash, you can backup Kubernetes volumes mounted in following types of workloads:
- Deployment
- DaemonSet
- ReplicaSet
- ReplicationController
- StatefulSet

## Features
 - Fast, secure, efficient backup of any kubernetes [volumes](https://kubernetes.io/docs/concepts/storage/volumes/).
 - Automates configuration of `restic` for periodic backup.
 - Store backed up files in various cloud storage provider, including S3, GCS, Azure, OpenStack Swift etc.
 - Prometheus ready metrics for backup process.

## Supported Versions
Kubernetes 1.5+

## Installation
To install Stash, please follow the guide [here](/docs/install.md).

## Using Stash
Want to learn how to use Stash? Please start [here](/docs/tutorial.md).

## Contribution guidelines
Want to help improve Stash? Please start [here](/CONTRIBUTING.md).

## Project Status
Wondering what features are coming next? Please visit [here](/ROADMAP.md).

---

**The stash operator collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--analytics=false`.

---

## Acknowledgement
 - Many thanks to [Alexander Neumann](https://github.com/fd0) for [Restic](https://github.com/restic/restic) project.

## Support
If you have any questions, you can reach out to us.
* [Slack](https://slack.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
* [Website](https://appscode.com)

[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/stash)](https://goreportcard.com/report/github.com/appscode/stash)

# Stash
 Stash by AppsCode is a Kubernetes operator for [restic](https://restic.net). If you are running production workloads in Kubernetes, you might want to take backup of your disks. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. `restic` is a backup program that is fast, efficient and secure with few moving parts. Stash is a CRD controller for Kubernetes built around `restic` to address these issues. Using Stash, you can backup Kubernetes volumes mounted in following types of workloads:
- Deployment
- DaemonSet
- ReplicaSet
- ReplicationController
- StatefulSet

## Features
 - Fast, secure, efficient backup of any kubernetes [volumes](https://kubernetes.io/docs/concepts/storage/volumes/).
 - Automates configuration of `restic` for periodic backup.
 - Store backed up files in various cloud storage provider, including S3, GCS, Azure, OpenStack Swift, DigitalOcean Spaces etc.
 - Restore backup easily.
 - Periodically check integrity of backed up data.
 - Take backup in offline mode.
 - Support workload initializer for faster backup.
 - Prometheus ready metrics for backup process.

## Supported Versions
Please pick a version of Stash that matches your Kubernetes installation.

| Stash Version                                                            | Docs                                                            | Kubernetes Version |
|--------------------------------------------------------------------------|-----------------------------------------------------------------|--------------------|
| [0.6.3](https://github.com/appscode/stash/releases/tag/0.6.3) (uses CRD) | [User Guide](https://appscode.com/products/stash/0.6.3)         | 1.7.x+             |
| [0.4.2](https://github.com/appscode/stash/releases/tag/0.4.2) (uses TPR) | [User Guide](https://github.com/appscode/stash/tree/0.4.2/docs) | 1.5.x - 1.7.x      |

## Installation

To install Stash, please follow the guide [here](https://appscode.com/products/stash/0.6.3/setup/install).

## Using Stash
Want to learn how to use Stash? Please start [here](https://appscode.com/products/stash/0.6.3).

## Contribution guidelines
Want to help improve Stash? Please start [here](https://appscode.com/products/stash/0.6.3/welcome/contributing).

---

**Stash binaries collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--analytics=false`.

---

## Acknowledgement
 - Many thanks to [Alexander Neumann](https://github.com/fd0) for [Restic](https://restic.net) project.

## Support
We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [AppsCode Slack team](https://appscode.slack.com/messages/C8NCX6N23/details/) channel `#stash`. To sign up, use our [Slack inviter](https://slack.appscode.com/).

If you have found a bug with Searchlight or want to request for new features, please [file an issue](https://github.com/appscode/stash/issues/new).

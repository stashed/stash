[![Go Report Card](https://goreportcard.com/badge/stash.appscode.dev/stash)](https://goreportcard.com/report/stash.appscode.dev/stash)
[![Build Status](https://github.com/stashed/stash/workflows/CI/badge.svg)](https://github.com/stashed/stash/actions?workflow=CI)
[![codecov](https://codecov.io/gh/stashed/stash/branch/master/graph/badge.svg)](https://codecov.io/gh/stashed/stash)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/stash.svg)](https://hub.docker.com/r/appscode/stash/)
[![Slack](https://slack.appscode.com/badge.svg)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/kubestash.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=KubeStash)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fstashed%2Fstash.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fstashed%2Fstash?ref=badge_shield)

# Stash

[Stash](https://stash.run) by AppsCode is a cloud native data backup and recovery solution for Kubernetes workloads. If you are running production workloads in Kubernetes, you might want to take backup of your disks, databases etc. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. Stash is a Kubernetes operator that uses [restic](https://github.com/restic/restic) or Kubernetes CSI Driver VolumeSnapshotter functionality to address these issues. Using Stash, you can backup Kubernetes volumes mounted in workloads, stand-alone volumes and databases. User may even extend Stash via [addons](https://stash.run/docs/latest/guides/latest/addons/overview/) for any custom workload.

## Features

| Features                                                                        | Availability | Scope                                                                                                                                                 |
| ------------------------------------------------------------------------------- | :----------: | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Backup & Restore Workload Data                                                  |   &#10003;   | Deployment, DaemonSet, StatefulSet, ReplicaSet, ReplicationController, OpenShift DeploymentConfig                                                     |
| Backup & Restore Stand-alone Volume (PVC)                                       |   &#10003;   | PersistentVolumeClaim, PersistentVolume                                                                                                               |
| Backup & Restore databases                                                      |   &#10003;   | PostgreSQL, MySQL, MongoDB, Elasticsearch                                                                                                             |
| [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/) |   &#10003;   | CSI Driver must support VolumeSnapshot and Kubernetes Alpha features must be enabled                                                                  |
| Schedule Backup                                                                 |   &#10003;   | Schedule through [cron expression](https://en.wikipedia.org/wiki/Cron)                                                                                |
| Instant Backup                                                                  |   &#10003;   | Use CLI or create BackupSession manually                                                                                                              |
| Auto Backup                                                                     |   &#10003;   | Using a Template and annotations                                                                                                                      |
| Batch Backup                                                                    |   &#10003;   | Backup multiple co-related targets under a single configuration                                                                                       |
| Pause Scheduled Backup                                                          |   &#10003;   |                                                                                                                                                       |
| Support Multiple Storage Provider                                               |   &#10003;   | AWS S3, Minio, Rook, GCS, Azure, OpenStack Swift,  Backblaze B2, Rest Server, any PV/PVC                                                              |
| Encryption                                                                      |   &#10003;   | AES-256 in counter mode (CTR) (for Restic driver)                                                                                                     |
| Deduplication (send only diff)                                                  |   &#10003;   | Uses [Content Defined Chunking (CDC)](https://restic.net/blog/2015-09-12/restic-foundation1-cdc) (for Restic driver)                                  |
| Cleanup old snapshots automatically                                             |   &#10003;   | Cleanup according to different [retention policies](https://restic.readthedocs.io/en/stable/060_forget.html#removing-snapshots-according-to-a-policy) |
| Prometheus Metrics for Backup & Restore Process                                 |   &#10003;   | Official Prometheus Server, CoreOS Prometheus Operator                                                                                                |
| Prometheus Metrics for Stash Operator                                           |   &#10003;   | Official Prometheus Server, CoreOS Prometheus Operator                                                                                                |
| Support RBAC enabled cluster                                                    |   &#10003;   |                                                                                                                                                       |
| Support PSP enabled cluster                                                     |   &#10003;   |                                                                                                                                                       |
| CLI                                                                             |   &#10003;   | `kubectl` plugin (for Kubernetes 1.12+)                                                                                                               |
| Extensibility                                                                   |   &#10003;   | Extend using `Function` and `Task`                                                                                                                    |
| Customizability                                                                 |   &#10003;   | Customize backup / restore process using `Function` and `Task`                                                                                        |
| Hooks                                                                           |   &#10003;   | Execute `httpGet`, `httpPost`, `tcpSocket` and `exec` hooks before and after  of backup or restore process.                                           |
| Send Notification to Webhook                                                    |   &#10003;   | Use hooks to send notification to webhooks(i.e. Slack channel)                                                                                        |

## Supported Versions

Please pick a version of Stash that matches your Kubernetes installation.

| Stash Version                                                                       | Docs                                                          | Kubernetes Version |
| ----------------------------------------------------------------------------------- | ------------------------------------------------------------- | ------------------ |
| [v0.9.0-rc.6](https://github.com/stashed/stash/releases/tag/v0.9.0-rc.6) (uses CRD) | [User Guide](https://stash.run/docs/v0.9.0-rc.6)              | 1.11.x+            |
| [0.8.3](https://github.com/stashed/stash/releases/tag/0.8.3) (uses CRD)             | [User Guide](https://stash.run/docs/0.8.3)                    | 1.9.x+             |
| [0.7.0](https://github.com/stashed/stash/releases/tag/0.7.0) (uses CRD)             | [User Guide](https://stash.run/docs/0.7.0)                    | 1.8.x              |
| [0.6.4](https://github.com/stashed/stash/releases/tag/0.6.4) (uses CRD)             | [User Guide](https://stash.run/docs/0.6.4)                    | 1.7.x              |
| [0.4.2](https://github.com/stashed/stash/releases/tag/0.4.2) (uses TPR)             | [User Guide](https://github.com/stashed/docs/tree/0.4.2/docs) | 1.5.x - 1.6.x      |

## Installation

To install Stash, please follow the guide [here](https://stash.run/docs/latest/setup/install).

## Using Stash

Want to learn how to use Stash? Please start [here](https://stash.run/docs/latest/).

## Stash API Clients

You can use Stash api clients to programmatically access its objects. Here are the supported clients:

- Go: [https://github.com/stashed/stash](/client/clientset/versioned)
- Java: https://github.com/stashed/java

## Contribution guidelines

Want to help improve Stash? Please start [here](https://stash.run/docs/latest/welcome/contributing).

---

**Stash binaries collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--enable-analytics=false`.

---

## Acknowledgement

- Many thanks to [Alexander Neumann](https://github.com/fd0) for [Restic](https://restic.net) project.

## Support

We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [AppsCode Slack team](https://appscode.slack.com/messages/C8NCX6N23/details/) channel `#stash`. To sign up, use our [Slack inviter](https://slack.appscode.com/).

If you have found a bug with Stash or want to request for new features, please [file an issue](https://github.com/stashed/stash/issues/new).

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fstashed%2Fstash.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fstashed%2Fstash?ref=badge_large)

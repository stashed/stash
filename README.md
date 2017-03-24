

## Restik
 Restik provides support to backup your Kubernetes Volumes

**Feautures**
 - Support backup for any kubernetes [volumes](https://kubernetes.io/docs/concepts/storage/volumes/).
## Supported Versions
Kubernetes 1.5+

## Supported Workloads 
Restik supports backup of following Workloads

* Replication Controller
* Replica Set 
* Deployment
* Daemon Set
* Stateful Set

## Installation
Installation and Upgrade process are described [here](docs/user-guide/install.md)

## How to backup

One can start the backup process by following this [guide](docs/user-guide/backup.md)

## How to recover

The recover process will be left to users for now. It can be done by running `$ /restic -r <restik_repo> restore snapshot_id --target <target_dir>` inside the restic-sidecar container. 
You can find the details [here](https://restic.readthedocs.io/en/stable/Manual/#restore-a-snapshot) 

## Developer Guide
Want to learn whats happening under the hood, read [the developer guide](docs/developer-guide/README.md).

## Architectural Design
If you want to know how Backup Controller is working, read this [doc](docs/developer-guide/design.md).

## Contribution
If you're interested in being a contributor, read [the contribution guide](docs/contribution/README.md).

## Acknowledgement
 - Restic https://github.com/restic/restic
 
## Support
If you have any questions, you can reach out to us.
[Website](https://appscode.com) • [Slack](https://slack.appscode.com) • [Forum](https://discuss.appscode.com) • [Twitter](https://twitter.com/AppsCodeHQ)
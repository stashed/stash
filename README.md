[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/restik)](https://goreportcard.com/report/github.com/appscode/restik)

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

## Versioning Policy
There are 2 parts to versioning policy:
 - Operator version: Restik __does not follow semver__, rather the _major.minor_ version of operator points to the
Kubernetes client version. You can verify this from the `glide.yaml` file. This means there might be breaking changes
between point releases of the operator. This generally manifests as changed annotation keys or their meaning.
Please always check the release notes for upgrade instructions.
 - TPR version: backup.appscode.com/v1alpha1 is considered in alpha. This means breaking changes to the YAML format
might happen among different releases of the operator.

---

**The restik operator collects anonymous usage statistics to help us learning
how the software is being used and how we can improve it. To disable stats collection,
run the operator with the flag** `--analytics=false`.

---

## Contribution Guidelines
If you're interested in being a contributor, read [the contribution guide](docs/contribution/README.md).

## Acknowledgement
 - Restic https://github.com/restic/restic

## Support
If you have any questions, you can reach out to us.

* [Slack](https://slack.appscode.com)
* [Forum](https://discuss.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
* [Website](https://appscode.com)

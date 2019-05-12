---
title: Stash Architecture
description: Stash Architecture
menu:
  product_stash_0.8.3:
    identifier: architecture-concepts
    name: Architecture
    parent: what-is-stash
    weight: 20
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---

# Stash Architecture

Stash is a Kubernetes operator for [restic](https://restic.net/). At the heart of Stash, it is a Kubernetes [controller](https://book.kubebuilder.io/basics/what_is_a_controller.html). It uses [Custom Resource Definition(CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to specify targets and behaviors of backup and restore process in a Kubernetes native way. A simplified architecture of Stash is shown below:

<figure align="center">
  <img alt="Stash Architecture" src="/docs/images/concepts/stash_architecture.svg">
  <figcaption align="center">Fig: Stash Architecture</figcaption>
</figure>

## Components

Stash consists of various components that implement backup and restore logic. This section will give you a brief overview of such components.

### Stash Operator

When a user installs Stash, it creates a Kubernetes [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) typically named `stash-operator`. This deployment controls the entire backup and restore process. `stash-operator` deployment runs two containers. One of them is called `operator` which performs the core functionality of Stash and the other one is `pushgateway` which is a Prometheus [pushgateway](https://github.com/prometheus/pushgateway).

#### Operator

`operator` container runs all the controllers as well as an [Aggregated API Server](https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/).

##### Controllers

Controllers watch various Kubernetes resources as well as the custom resources introduced by Stash. It applies the backup or restore logic for a target resource when requested by users.

##### Aggregated API Server

Aggregated API Server self-hosts validating and mutating [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) and runs an Extension API Server for Snapshot resource.

- **Mutating Webhook:** Stash uses Mutating Webhook to inject backup `sidecar` or restore `init-container` into a workload if any backup or restore process is configured for it. It is also used for defaulting custom resources.

- **Validating Webhook:** Validating Webhook is used to validate the custom resource objects.

- **Snapshot Server:** Stash uses Kubernetes Extended API Server to provide `view` and `list` capability of backed up snapshots. When a user requests for Snapshot objects, Snapshot server reads respective information directly from backend repository and returns object representation in a Kubernetes native way.

#### Pushgateway

`pushgateway` container runs Prometheus [pushgateway](https://github.com/prometheus/pushgateway). All the backup sidecars/jobs and restore init-containers/jobs send Prometheus metrics to this pushgateway after completing their backup or restore process. Prometheus server can scrap those metrics from this pushgateway.

### Backend

Backend is the storage where Stash stores backed up files. It can be a cloud storage like GCS bucket, AWS S3, Azure Blob Storage etc. or a Kubernetes persistent volume like [NFS](https://kubernetes.io/docs/concepts/storage/volumes/#nfs), [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim), etc. To learn more about backend, please visit [here](/docs/guides/backends/overview.md).

### CronJob

When a user creates a [BackupConfiguration](#backupconfiguration) object, Stash creates a CronJob with the schedule specified in it. At each scheduled slot, this CronJob triggers a backup for the targeted workload.

### Backup Sidecar / Backup Job

When an user creates a [BackupConfiguration](#backupconfiguration) object, Stash injects a `sidecar` to the target if it is a workload (i.e. `Deployment`, `DaemonSet`, `StatefulSet` etc.). This `sidecar` takes backup when the respective CronJob triggers a backup. If the target is a database or stand-alone volume, Stash creates a job to take backup at each trigger.

### Restore Init-Container / Restore Job

When an user creates a [RestoreSession](#restoresession) object, Stash injects an `init-container` to the target if it is a workload (i.e. `Deployment`, `DaemonSet`, `StatefulSet` etc.). This `init-container` performs restore process on restart. If the target is a database or stand-alone volume, Stash creates a job to restore the target.

### Custom Resources

Stash uses [Custom Resource Definition(CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to specify targets and behaviors of backup and restore process in a Kubernetes native way. This section will give you a brief overview of the custom resources used by Stash.

- **Repository**

  A `Repository` specifies the backend storage system where the backed up data will be stored. A user has to create `Repository` object for each backup target. Only one target can be backed up into one `Repository`. For details about `Repository`, please visit [here](/docs/concepts/crds/repository.md).

- **BackupConfiguration**

  A `BackupConfiguration` specifies the backup target, behaviors (schedule, retention policy etc.), `Repository` object that holds backend information etc. A user has to create one `BackupConfiguration` object for each backup target. When a user creates a `BackupConfiguration`, Stash creates a CronJob for it and injects backup sidecar to the target if it is a workload (i.e. Deployment, DaemonSet, StatefulSet etc.). For more details about `BackupConfiguration`, please visit [here](/docs/concepts/crds/backupconfiguration.md).

- **BackupSession**

  A `BackupSession` object represents a backup run of a target. It is created by respective CronJob at each scheduled time slot. It refers to a `BackupConfiguration` object for necessary configuration. Controller that runs inside backup sidecar (in case of backup via job, it is stash operator itself) will watch this `BackupSession` object and start taking the backup instantly. A user can also create a `BackupSession` object manually to trigger instant backups. For more details about `BackupSession`s, please visit [here](/docs/concepts/crds/backupsession.md).

- **RestoreSession**

  A `RestoreSession` specifies what to restore and the source of data. A user has to create a `RestoreSession` object when s/he wants to restore a target. When s/he creates a `RestoreSession`, Stash injects an `init-container` into the target workload (launches a job if the target is not a workload) to restore. For more details about `RestoreSession`, please visit [here](/docs/concepts/crds/restoresession.md).

- **Function**

  A `Function` is a template for a container that performs only a specific action.  For example, `pg-backup` function only dumps and uploads the dumped file into the backend, whereas `update-status` function updates the status of the respective `BackupSession` and `Repository` and sends Prometheus metrics to `pushgateway` based on the output of another function. For more details about `Function`, please visit [here](/docs/concepts/crds/function.md).

- **Task**

  A complete backup or restore process may consist of several steps. For example, in order to backup a PostgreSQL database we first need to dump the database, upload the dumped file to backend and then we need to update `Repository` and `BackupSession` status and send Prometheus metrics. We represent such individual steps via `Function` objects. An entire backup or restore process needs an ordered execution of one or more functions. A `Task` specifies an ordered collection of functions along with their parameters. `Function` and `Task` enables users to extend or customize the backup/restore process. For more details about `Task`, please visit [here](/docs/concepts/crds/task.md).

- **BackupConfigurationTemplate**

  A `BackupConfigurationTemplate` enables users to provide a template for `Repository` and `BackupConfiguration` object. Then, s/he just needs to add some annotations to the workload s/he wants to backup. Stash will automatically create respective `Repository` and `BackupConfiguration` according to the template. In this way, users can create a single template for all similar types of workloads and backup them by applying some annotations on them. In Stash parlance, we call this process **default backup**. For more details about `BackupConfigurationTemplate`, please visit [here](/docs/concepts/crds/backupconfiguration_template.md).

- **AppBinding**

  An `AppBinding` holds necessary information to connect with a database. For more details about `AppBinding`, please visit [here](/docs/concepts/crds/appbinding.md).

- **Snapshot**

  A `Snapshot` is a representation of a backup snapshot in a Kubernetes native way. Stash uses Kuberentes Extended API Server for handling `Snapshot`s. For more details about `Snapshot`s, please visit [here](/docs/concepts/crds/snapshot.md).

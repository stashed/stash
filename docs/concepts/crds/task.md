---
title: Task Overview
menu:
  product_stash_0.8.3:
    identifier: task-overview
    name: Task
    parent: crds
    weight: 35
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: concepts
---

> New to Stash? Please start [here](/docs/concepts/README.md).

# Task

## What is Task

An entire backup or restore process needs an ordered execution of one or more steps. A [Function](/docs/concepts/crds/function.md) represents a step of a backup or restore process. A `Task` is a Kubernetes `CustomResourceDefinition`(CRD) which specifies a sequence of functions along with their parameters in a Kubernetes native way.

When you install Stash, some `Task`s will be pre-installed for supported targets like databases, etc. However, you can create your own `Task` to customize or extend the backup/restore process. Stash will execute these steps in the order you have specified.

## Task CRD Specification

Like any official Kubernetes resource, a `Task` has `TypeMeta`, `ObjectMeta` and `Spec` sections. However, unlike other Kubernetes resources, it does not have a `Status` section.

A sample `Task` object to backup a PostgreSQL database is shown below:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: Task
metadata:
  name: pg-backup
spec:
  steps:
  - name: pg-backup
    params:
    - name: outputDir # specifies where to write the output file
      value: /tmp/output
    - name: secretVolume # specifies where backend secret has been mounted
      value: secret-volume
  - name: update-status
    params:
    - name: outputDir # specifies where the previous step wrote the output file. it will read that file and update the status of respective resources accordingly.
      value: /tmp/output
  volumes:
  - name: secret-volume
    secret:
      secretName: ${REPOSITORY_SECRET_NAME}
```

This `Task` uses two functions to backup a PostgreSQL database. The first step indicates `pg-backup` function that dumps PostgreSQL database and uploads the dumped file. The second step indicates `update-status` function which updates the status of the `BackupSession` and `Repository` crd for respective backup.

Here, we are going to describe the various sections of a `Task` crd.

### Task `Spec`

A `Task` object has the following fields in the `spec` section:

#### spec.steps

`spec.steps` section specifies a list of functions and their parameters in the order they should be executed. You can also templatize this section using the [variables](/docs/concepts/crds/functions.md#stash-provided-variables) that Stash can resolve itself. Stash will resolve all the variables and create a pod definition with a container specification for each `Function` specified in `steps` section.

Each `step` consists of the following fields:

- **name :** `name` specifies the name of the `Function` that will be executed at this step.
- **params :** `params` specifies an optional list of variables names and their values that Stash should use to resolve the respective `Function`. If you use a variable in a `Function` specification whose value Stash cannot provide, you can pass the value of that variable using this `params` section. You have to specify the following fields for a variable:
  - **name :** `name` of the variable.
  - **value :** value of the variable.

In the above example `Task`, we have used `outputDir` variable in `pg-backup` function that Stash can't resolve automatically. So, we have passed the value using the `params` section in the `Task` object.

>Stash executes the `Functions` in the order they appear in `spec.steps` section. All the functions except the last one will be used to create `init-container` specification and the last function will be used to create `container` specification for respective backup job. This guarantees an ordered execution of the steps.

#### spec.volumes

`spec.volumes` specifies a list of volumes that should be mounted in the respective job created for this `Task`. In the sample we have shown above, we need to mount storage secret for the backup job. So, we have added the secret volume in `spec.volumes` section. Note that, we have used `REPOSITORY_SECRET_NAME` variable as secret name. This variable will be resolved by Stash from `Repository` specification.

## Why Function and Task?

You might be wondering why we have introduced `Function` and `Task` crd. We have designed `Function-Task` model for the following reasons:

- **Customizability:** `Function` and `Task` enables you to customize backup/recovery process. For example, currently we use [mysqldump](https://dev.mysql.com/doc/refman/8.0/en/mysqldump.html) in `mysql-backup` Function to backup MySQL database. You can build a custom `Function` using Percona's [xtrabackup](https://www.percona.com/software/mysql-database/percona-xtrabackup) tool instead of `mysqldump`. Then you can write a `Task` with this custom `Function` and use it to backup your target MySQL database.

You can also customize backup/restore process by executing hooks before or after the backup/restore process. For example, if you want to execute some logic to prepare your apps for backup or you want to send an email notification after each backup, you just need to add `Function` with your custom logic and respective `Task` to execute them.

- **Extensibility:** Currently, Stash supports backup of MySQL, MongoDB and PostgreSQL databases. You can easily backup the databases that are not officially supported by Stash. You just need to create a `Function` and a `Task` for your desired database.

- **Re-usability:** `Function`'s are self-sufficient and independent of Stash. So, you can reuse them in any application that uses `Function-Task` model.

## Next Steps

- Learn how Stash backup databases using `Function-Task` model from [here](/docs/guides/databases/backup.md).
- Learn how Stash backup stand-alone PVC using `Function-Task` model from [here](/docs/guides/volumes/backup.md).

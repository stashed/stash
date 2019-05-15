---
title: File Ownership | Stash
description: Handling Restored File Ownership in Stash
menu:
  product_stash_0.8.3:
    identifier: file-ownership-stash
    name: File Ownership
    parent: latest-guides
    weight: 150
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

# Handling Restored File Ownership in Stash

Stash preserves permission bits of the restored files. However, it may change ownership (owner `uid` and `gid`) of the restored files in some cases. This tutorial will explain when and how ownership of the restored files can be changed. Then, we will explain how we can avoid or resolve this problem.

## Understanding Backup and Restore Behaviour

At first, let's understand how backup and restore behaves in different scenario. A table with some possible backup and restore scenario is given below. We have run the containers as different user in different scenario using [SecurityContext](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/).

| Case  | Original File Owner | Backup `sidecar` User | Backup Succeed? | Restore `init-container` User |     Restore Succeed?      | Restored File Owner | Restored File Editable to Original Owner? |
| :---: | :-----------------: | :-------------------: | :-------------: | :---------------------------: | :-----------------------: | :-----------------: | :---------------------------------------: |
|   1   |        root         |     stash (1005)      |    &#10003;     |          stash(1005)          |         &#10003;          |        1005         |                 &#10003;                  |
|   2   |        2000         |      stash(1005)      |    &#10003;     |          stash(1005)          |         &#10003;          |        1005         |                 &#10007;                  |
|   3   |        2000         |         root          |    &#10003;     |             root              |         &#10003;          |        2000         |                 &#10003;                  |
|   4   |        2000         |         root          |    &#10003;     |          stash(1005)          | &#10003; (remote backend) |        1005         |                 &#10007;                  |
|   5   |        2000         |         root          |    &#10003;     |          stash(1005)          | &#10007; (local backend)  |          -          |                     -                     |
|   6   |        2000         |         3000          |    &#10003;     |          stash(1005)          | &#10003; (remote backend) |        1005         |                 &#10007;                  |
|   7   |        2000         |         3000          |    &#10003;     |          stash(1005)          | &#10007; (local backend)  |          -          |                     -                     |
|   8   |        2000         |         3000          |    &#10003;     |             root              |         &#10003;          |        2000         |                 &#10003;                  |
|   9   |        2000         |         3000          |    &#10003;     |             3000              |         &#10003;          |        3000         |                 &#10007;                  |

If we look at the table carefully, we will notice the following behaviors:

1. The user of the backup `sidecar` does not have any effect on backup. It just needs read permission of the target files.
2. If the restore container runs as `root` user then original ownership of the restored files are preserved.
3. If the restore container runs as `non-root` user then the ownership of the restored files is changed to restore container's user and restored files become read-only to the original user unless it was `root` user.

So, we can see when we run restore container as `non-root` user, it raises some serious concerns as the restored files become read-only to the original user. Next section will discuss how we can avoid or fix this problem.

## Avoid or Fix Ownership Issue

As we have seen when the ownership of the restored files gets changed, they can be unusable to the original user. We need to avoid or fix this issue.

There could be two scenarios for the restored files user.

1. Restored files user is the same as the original user.
2. Restored files user is different than the original user.

### Restored files user is the same as the original user

This is likely to be the most common scenario. Generally, the same application whose data was backed up will use the restored files after a restore. In this case, if your cluster supports running containers as `root` user, then it is fairly easy to avoid this issue. You just need to run the restore container as `root` user which Stash does by default. However, things get little more complicated when your cluster does not support running containers as `root` user. In this case, you can do followings:

- Run restore container as the same user as the original container.
- Change the ownership of the restored files using `chown` after the restore is completed.

For the first method, we can achieve this configuring SecurityContext under `RuntimeSetting` of `RestoreSession` object. A sample `RestoreSession` objects configured SecurityContext to run as the same user as the original user (let original user is 2000) is shown below.

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: deployment-restore
  namespace: demo
spec:
  repository:
    name: local-repo
  rules:
  - paths:
    - /source/data
  target:
    ref:
      apiVersion: apps/v1
      kind: Deployment
      name: stash-demo
    volumeMounts:
    - name:  source-data
      mountPath:  /source/data
  runtimeSettings:
    container:
      securityContext:
        runAsUser: 2000
        runAsGroup: 2000
```

>If you are using [local](/docs/guides/latest/backends/local.md) backend to store backed up data, then the original container, backup container and restore container all of them must be run as same user. By default, Stash runs backup container as `stash(1005)` user. You can change this to match with the user of your original container using `securityContext` field under `runtimeSettings` of `BackupConfiguration` object.

The second method is necessary when the backup container was not run as the same user as the original container. This is similar to the process when the restored files user is different than the original user. In this case, you have to change the ownership of restored files using `chown` after restore.

### Restored file user is different than the original user

If you want to use the restore files with a different user than the original one, then you have to change the ownership after restore. You can use an `init-container` in your workload that will run `chown` command to change the permissions to the desired owner or you can exec into workload pod to change ownership yourself.

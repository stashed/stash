# Stash Design Overview

We are going to make a design overhaul of Stash to simplify backup and recovery process and support some most requested features. This doc will discuss what features stash is going to support and how these features may work.

We  have introduced some new crd  such as [Function](#function), [Task](#action) etc. and made whole process more modular. This will make easy to add support for new features and the users will also be able to customize backup process. Furthermore, this will make stash resources inter-operable between different tools and even might allow to use stash resources as function in serverless concept.

**We are hoping this design will graduate to GA. So, we are taking security seriously. We are going to make sure that nobody can bypass clusters security using Stash. This might requires to remove some existing features (for example, restore from different namespace). However, we will provide an alternate way to cover those use cases.**

## Goal

Goal of this new design to support following features:
- [Schedule Backup and Restore Workload Data](#schedule-backup-and-restore-workload-data)
- [Schedule Backup and Restore PVC](#schedule-backup-and-restore-pvc)
- [Schedule Backup and Restore Database](#schedule-backup-and-restore-database)
- [Schedule Backup Cluster YAMLs](#schedule-backup-cluster-yamls)
- [Trigger Backup Instantly](#trigger-backup-instantly)
- [Default Backup](#default-backup)
- [Auto Restore](#auto-restore)
- [Stash cli/kubectl-plugin](#stash-clikubectl-plugin)
- [Function](#function)
- [Task](#task)

## Schedule Backup and Restore Workload Data

### Backup Workload Data

User will be able to backup data from  a running workload.

**What user have to do?**

- Create a `Repository` crd.
- Create a `BackupConfiguration` crd pointing to targeted workload.

Sample `Repository` crd:

```yaml
apiVersion: stash.appscode.com/v1alpha2
kind: Repository
metadata:
  name: stash-backup-repo
  namespace: demo
spec:
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: default/deployment/stash-demo
    storageSecretName: gcs-secret
```

Sample `BackupConfiguration` crd:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: workload-data-backup
  namespace: demo
spec:
  schedule: '@every 1h'
  # <no backupProcedure required for sidecar model>
  # repository refers to the Repository crd that hold backend information
  repository:
    name: stash-backup-repo
  # target indicate the target workload that we want to backup
  target:
    ref:
      apiVersion: apps/v1
      kind: Deployment
      name: stash-demo
    # directories indicates the directories inside the workload we want to backup
    directories:
    - /source/data
  # retentionPolicies specify the policy to follow to clean old backup snapshots
  retentionPolicy:
    keepLast: 5
    prune: true
```

**How it will work?**

- Stash will watch for `BackupCofiguration` crd. When it will find a  `BackupConfiguration` crd, it will inject a `sidecar` container to the workload and start a `cron` for scheduled backup.
- In each schedule, the `cron` will create `BackupSession` crd.
- The `sidecar` container watches for `BackupSession` crd. If it find one, it will take backup instantly and update `BackupSession` status accordingly.

Sample `BackupSession` crd:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupSession
metadata:
  name: demo-volume-backup-session
  namespace: demo
spec:
  # backupConfiguration indicates the BackupConfiguration crd of respective target that we want to backup
  backupConfiguration:
    name: backup-volume-demo
status:
  observedGeneration: 239844#2
  phase: Succeed
  stats:
  - direcotry: /source/data
    snapshot: 40dc1520
    size: 1.720 GiB
    uploaded: 1.200 GiB # upload size can be smaller than original file size if there are some duplicate files
    fileStats:
      new: 5307
      changed: 0
      unmodified: 0
```

### Restore Workload Data

User will be able to restore backed up data  either into a separate volume or into the same workload from where the backup was taken. Here, is an example for recovering into same workload.

**What user have to do?**

- Create a `RestoreSession` crd pointing `target` field to the workload.

Sample `RestoreSession` crd to restore into same workload:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: recovery-database-demo
  namespace: demo
spec:
  repository:
    name: stash-backup-repo
  target: # target indicates where the recovered data will be stored
    ref:
      apiVersion: apps/v1
      kind: Deployment
      name: stash-demo
    directories: # indicates which directories will be recovered
    - /source/data
```

**How it will work?**

- When Stash will find a `RestoreSession` crd created to restore into a workload, it will inject a `init-container` to the targeted workload.
- Then, it will restart the workload.
- The `init-container` will restore data inside the workload.

> **Warning:** Restore in same workload require to restart the workload. So, there will be downtime of the workload.

## Schedule Backup and Restore PVC

### Backup PVC

User will be also able to backup stand-alone pvc. This is useful for `ReadOnlyMany` or `ReadWriteMany` type pvc.

**What user have to do?**

- Create a `Repository` crd for respective backend.

- Create a `BackupConfiguration` crd pointing `target` field to the volume.

Sample `BackupConfiguration` crd to backup a PVC:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: volume-backup-demo
  namespace: demo
spec:
  schedule: '@every 1h'
  # task indicates Task crd that specifies the steps to backup a volume.
  # stash will create some default Task crd  while install to backup/restore various resources.
  # user can also crate their own Task to customize backup/recovery
  task:
    name: volumeBackup
  # repository refers to the Repository crd that hold backend information
  repository:
    name: stash-backup-repo
  # target indicate the target workload that we want to backup
  target:
    ref:
      apiVersion: v1
      kind: PersistentVolumeClaim
      name: demo-pvc  
    mountPath: /source/data
  # retentionPolicies specify the policy to follow to clean old backup snapshots
  retentionPolicy:
    keepLast: 5
    prune: true
```

**How it will work?**

1. Stash will create a `CronJob` using information of respective `Task` crd specified by `task` field.
2. The `CronJob` will take periodic backup of the target volume.

### Restore PVC

User will be able to restore backed up data  into a  volume.

**What user have to do?**

- Create a `RestoreSession` crd pointing `target` field to the target volume where the recovered data will be stored.

Sample `RestoreSession` crd to restore into a volume:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: recovery-volume-demo
  namespace: demo
spec:
  repository:
    name: stash-backup-repo
  # task indicates Task crd that specifies steps to restore a volume
  task:
    name: volumeRecovery
  target: # target indicates where the recovered data will be stored
    ref:
      apiVersion: v1
      kind: PersistentVolumeClaim
      name: demo-pvc  
    mountPath: /source/data
    directories: # indicates which directories will be recovered
    - /source/data
```

**How it will work?**

- When Stash will find a `RestoreSession` crd created to restore into a volume, it will launch a Job to restore into that volume.
- The recovery Job will restore and store recovered data to the specified volume.

## Schedule Backup and Restore Database

### Backup Database

User will be able to backup database using Stash.

**What user have to do?**

- Create a `Repository` crd for respective backend.
- Create an `AppBinding` crd which holds connection information for the database. If the database is deployed with [KubeDB](https://kubedb.com/docs/0.9.0/welcome/), `AppBinding` crd will be created automatically for each database.
- Create a `BackupConfiguration` crd pointing to the `AppBinding` crd.

Sample `AppBinding` crd:

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  name: quick-postgres
  namespace: demo
  labels:
    kubedb.com/kind: Postgres
    kubedb.com/name: quick-postgres
spec:
  clientConfig:
    insecureSkipTLSVerify: true
    service:
      name: quick-postgres
      port: 5432
      scheme: "http"
  secret:
    name: quick-postgres-auth
  type: kubedb.com/postgres
```

Sample `BackupConfiguration` crd for database backup:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: database-backup-demo
  namespace: demo
spec:
  schedule: '@every 1h'
  # task indicates Task crd that specifies the steps to backup postgres database
  task:
    name:   pgBackup
    inputs:
      database: my-postgres # specify this field if you want to backup a particular database.
  # repository refers to the Repository crd that hold backend information
  repository:
    name: stash-backup-repo
  # target indicates the respective AppBinding crd for target database
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: quick-postgres
  # retentionPolicies specify the policy to follow to clean old backup snapshots
  retentionPolicy:
    keepLast: 5
    prune: true
```

**How it will work?**

- When Stash will see a `BackupConfiguration` crd for database backup, it will lunch  a `CronJob` to take periodic backup of this database.

### Restore Database

User will be able to initialize a database from backed up snapshot.

**What user have to do?**

- Create a `RestoreSession` crd with `target` field pointing to respective `AppBinding` crd of the target database.

Sample `RestoreSession` crd to restore database:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: database-recovery-demo
  namespace: demo
spec:
  repository:
    name: stash-backup-repo
  # task indicates Task crd that specifies the steps to restore Postgres database
  task:
    name: pgRecovery
  target: # target indicates where to restore
    # indicates the respective AppBinding crd for target database that we want to initialize from backup
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: quick-postgres
```

**How it will work?:**

- Stash will lunch a Job to restore the backed up database and initialize target with this recovered data.

## Schedule Backup Cluster YAMLs

User will be able to backup yaml of the cluster resources. However, currently stash will not provide automatic restore cluster from the YAMLs. So, user will have to create them manually.

In future, Stash might be able to backup and restore not only YAMLs but also entire cluster.

**What user have to do?**

- Create a `Repository` crd for respective backend.
- Create a `BackupConfiguration` crd with `task` field point to a `Task` crd that backup cluster.

Sample `BackupConfiguration` crd to backup YAMLs of cluster resources:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: cluster-backup-demo
  namespace: demo
spec:
  schedule: '@every 1h'
  # task indicates Task crd that specifies the steps of backup cluster yamls
  task:
    name: clusterBackup
  # repository refers to the Repository crd that hold backend information
  repository:
    name: stash-backup-repo
  # <no target required for cluster backup>
  # retentionPolicies specify the policy to follow to clean old backup snapshots
  retentionPolicy:
    keepLast: 5
    prune: true
```

**How it will work?**

- Stash will lunch a `CronJob` using informations of the `Task` crd specified through `task` filed.
- The `CronJob` will take periodic backup of the cluster.

## Trigger Backup Instantly

User will be able to trigger a scheduled backup instantly. 

**What user have to do?**

- Create a `BackupSession` crd pointing to the target `BackupConfiguration` crd.

Sample `BackupSession` crd for triggering instant backup:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupSession
metadata:
  name: demo-volume-backup-session
  namespace: demo
spec:
  # backupConfiguration indicates the BackupConfiguration crd of respective target that we want to backup
  backupConfiguration:
    name: volume-backup-demo
```

**How it will work?**

- For scheduled  backup through `sidecar` container, the `sidecar` container will take instant backup as it watches for `BackupSession` crd.
- For scheduled backup through `CronJob`, Stash will lunch another job to take instant backup of the target.

## Default Backup

User will also be able to configure a `default` backup for the cluster. So, user will no longer need to create  `Repository` and  `BackupConfiguration` crd for every workload he want to backup. Instead, she will need to add some annotations to the target workload.

**What user have to do?**

- Create a `BackupTemplate` crd which will hold backend information and backup information.
- Add some annotations to the target. If the target is a database then add the annotations to respective `AppBinding` crd.

### Default Backup of Workload Data

Sample `BackupTemplate` crd to backup workload data:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupTemplate
metadata:
  name: workload-data-backup-template
spec:
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: ${target.namespace}/${target.name} # this prefix template will be used to initialize repository in different directory in backend.
    storageSecretName: gcs-secret # users must ensure this secret is present in respective namespace
  schedule: '@every 1h'
  # < no task required >
  retentionPolicy:
    name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Sample  workload with annotations for default backup:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stash-demo
  namespace: demo
  labels:
    app: stash-demo
  # if stash find bellow annotations, it will take backup of it.
  annotations:
    stash.appscode.com/backuptemplate: "workload-data-backup-template"
    stash.appscode.com/targetDirectories: "[/source/data]"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: stash-demo
  template:
    metadata:
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - args:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      restartPolicy: Always
      volumes:
      - name: source-data
        configMap:
          name: stash-sample-data
```

### Default Backup of a PVC

Sample `BackupTemplate` crd for stand-alone pvc backup:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupTemplate
metadata:
  name: volume-backup-template
spec:
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: ${target.namespace}/${target.name} # this prefix template will be used to initialize repository in different directory in backend.
    storageSecretName: gcs-secret # users must ensure this secret is present in respective namespace
  schedule: '@every 1h'
  task:
    name: volumeBackup
  retentionPolicy:
    name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Sample PVC with annotation for default backup:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: demo-pvc
  namespace: demo
  # if stash find bellow annotations, it will take backup of it.
  annotations:
    stash.appscode.com/backuptemplate: "volume-backup-template"
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
```

### Default Backup of Database

Sample `BackupTemplate` crd for database backup:

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupTemplate
metadata:
  name: pgdb-backup-template
spec:
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: ${target.namespace}/${target.name} # this prefix template will be used to initialize repository in different directory in backend.
    storageSecretName: gcs-secret # users must ensure this secret is present in respective namespace
  schedule: '@every 1h'
  task:
    name: pgBackup
  retentionPolicy:
    name: 'keep-last-5'
    keepLast: 5
    prune: true
```

Sample `AppBinding` crd with annotations for default backup:

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  name: quick-postgres
  namespace: demo
  labels:
    kubedb.com/kind: Postgres
    kubedb.com/name: quick-postgres
    # if stash find bellow annotations, it will take backup of it.
    annotations:
      stash.appscode.com/backuptemplate: "pgdb-backup-template"
spec:
  clientConfig:
    insecureSkipTLSVerify: true
    service:
      name: quick-postgres
      port: 5432
      scheme: "http"
  secret:
    name: quick-postgres-auth
  type: kubedb.com/postgres
```

**How it will work?**

- Stash will watch the workloads, volume and `AppBinding` crds. When Stash will find an workload/volume/AppBinding crd with these annotations, it will create a `Repository` crd and a `BackupConfiguration` crd using the information from respective `Task`.
- Then, Stash will take normal backup as discussed earlier.

## Auto Restore

User will be also able to configure an automatic recovery for a particular workload. Each time the workload restart, at first it will perform restore data from backup then original workload's container will start.

**What user have to do?**

- User will have to provide some annotations in the workload.

Sample workload wit annotation to restore on restart:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stash-demo
  namespace: demo
  labels:
    app: stash-demo
  # This annotations indicates that data shoul be recovered on each restart of the workload
  annotations:
    stash.appscode.com/restorepolicy: "OnRestart"
    stash.appscode.com/repository: "demo-backup-repo"
    stash.appscode.com/directories: "[/source/data]"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: stash-demo
  template:
    metadata:
      labels:
        app: stash-demo
      name: busybox
    spec:
      containers:
      - args:
        - sleep
        - "3600"
        image: busybox
        imagePullPolicy: IfNotPresent
        name: busybox
        volumeMounts:
        - mountPath: /source/data
          name: source-data
      restartPolicy: Always
      volumes:
      - name: source-data
        configMap:
          name: stash-sample-data
```

**How it will work?**

- When Stash will see a `RestoreSession` crd configured for auto recovery, it will inject an `init-container` to the target.
- The `init-container` will perform recovery on each restart.

## Stash cli/kubectl-plugin

We are going to provide a Stash plugin for `kubectl`. This will help to perform following operations:

- Restore into local machine instead of cluster (necessary for testing purpose).
- Restore into a different namespace from a repository: copy repository + secret into the desired namespace and then create `RestoreSession` object.
- Backup PV: creates matching PVC from PV (ensures that user has permission to read PV)
- Trigger instant backup.

## Function

`Function` are independent single-containered workload specification that perform only single task. For example, [pgBackup](#pgbackup) takes backup a PostgreSQL database and [clusterBackup](#clusterbackup) takes backup of YAMLs of cluster resources. `Function` crd has some variable fields with `$` prefix which hast be resolved while creating respective workload. You can consider these variable fields as input for an `Function`.

Some example `Function` definition is given below:

#### clusterBackup

```yaml
# clusterBackup function backup yamls of all resources of the cluster
apiVersion: stash.appscode.com/v1beta1
kind: Function
metadata:
  name: clusterBackup
spec:
  container:
    image:  appscodeci/cluster-tool:v1
    name:  cluster-tool
    args:
    - backup
    - --sanitize=${sanitize}
    - --provider=${provider}
    - --hostname=${hostname}
    - --path=${repoDir}
    - --output-dir=${outputDir}
    - --retention-policy.policy=${policy}
    - --retention-policy.value=${retentionValue}
    - --metrics.enabled=${enableMetric}
    - --metrics.pushgateway-url=${pushgatewayURL}
    - --metrics.labels="workload-kind=${workloadKind},workload-name=${workloadName}"
    volumeMounts:
    - name: ${tempVolumeName}
      mountPath: /tmp/restic
    - name: ${storageSecretName}
      mountPath: /etc/secrets/storage-secret
```

#### pgBackup

```yaml
# pgBackup function backup a PostgreSQL database
apiVersion: stash.appscode.com/v1beta1
kind: Function
metadata:
  name: pgBackup
spec:
  container:
    image:  appscodeci/postgresql-tool:v1
    name:  postgres-tool
    args:
    - backup
    - --database=${databases}
    - --provider=${provider}
    - --hostname=${hostname}
    - --path=${repoDir}
    - --output-dir=${outputDir}
    - --retention-policy.policy=${policy}
    - --retention-policy.value=${retentionValue}
    - --metrics.enabled=${enableMetric}
    - --metrics.pushgateway-url=${pushgatewayURL}
    - --metrics.labels="workload-kind=${workloadKind},workload-name=${workloadName}"
    env:
    - name:  PGPASSWORD
      valueFrom:
        secretKeyRef:
          name: $(databaseSecret)
          key: "POSTGRES_PASSWORD"
    - name:  DB_USER
      valueFrom:
        secretKeyRef:
          name: $(databaseSecret)
          key: "POSTGRES_USER"
    - name:  DB_HOST
      value: $(host)
    volumeMounts:
    - name: ${tempVolumeName}
      mountPath: /tmp/restic
    - name: ${storageSecretName}
      mountPath: /etc/secrets/storage-secret

```

#### pgRecovery

```yaml
# pgRecovery function restore a PostgreSQL database
apiVersion: stash.appscode.com/v1beta1
kind: Function
metadata:
  name: pgRecovery
spec:
  container:
    image:  appscodeci/postgresql-tool:v1
    name:  postgres-tool
    args:
    - restore
    - --provider=${provider}
    - --hostname=${hostname}
    - --path=${repoDir}
    - --output-dir=${outputDir}
    - --metrics.enabled=${enableMetric}
    - --metrics.pushgateway-url=${pushgatewayURL}
    - --metrics.labels="workload-kind=${workloadKind},workload-name=${workloadName}"
    env:
    - name:  PGPASSWORD
      valueFrom:
        secretKeyRef:
          name: $(databaseSecret)
          key: "POSTGRES_PASSWORD"
    - name:  DB_USER
      valueFrom:
        secretKeyRef:
          name: $(databaseSecret)
          key: "POSTGRES_USER"
    - name:  DB_HOST
      value: $(host)
    volumeMounts:
    - name: ${tempVolumeName}
      mountPath: /tmp/restic
    - name: ${storageSecretName}
      mountPath: /etc/secrets/storage-secret
```

#### stashPostBackup

```yaml
# stashPostBackup update Repository and BackupSession status for respective backup
apiVersion: stash.appscode.com/v1beta1
kind: Function
metadata:
  name: stashPostBackup
spec:
  container:
    image: appscode/stash:0.9.0
    name:  stash-post-backup
    args:
    - post-backup-update
    - --repository=${repoName}
    - --backupsession=${backupSessionName}
    - --output-json-dir=${outputJsonDir}
    volumeMounts:
    - name: ${outputVolumeName}
      mountPath: /tmp/restic
```

## stashPostRecovery

```yaml
# stashPostRecovery update RestoreSession status for respective recovery
apiVersion: stash.appscode.com/v1beta1
kind: Function
metadata:
  name: stashPostRecovery
spec:
  container:
    image: appscode/stash:0.9.0
    name:  stash-post-recovery
    args:
    - post-recovery-update
    - --recoveryconfiguration=${recoveryConfigurationName}
    - --output-json-dir=${outputJsonDir}
    volumeMounts:
    - name: ${outputVolumeName}
      mountPath: /tmp/restic
```

## Task

A complete backup process may need to perform multiple function. For example, if you want to backup a PostgreSQL database, we need to initialize a `Repository`, then backup the database and finally update `Repository` and `BackupSession` status to inform backup is completed or push backup metrics to a `pushgateway` . `Task` specifies these functions sequentially along with their inputs.

We have chosen to break complete backup process into several independent steps so that those individual functions can be used with other tool than Stash. It also make easy to add support for new feature. For example, to add support new database backup, we will just require to add a `Function` and `Task` crd. We will no longer need change anything in Stash operator code. This will also helps users to backup databases that are not officially supported by stash.

Some sample `Task` is given below:

#### pgBackup

```yaml
# pgBackup specifies required functions and their inputs to backup PostgreSQL database
apiVersion: stash.appscode.com/v1beta1
kind: Task
metadata:
  name: pgBackup
spec:
  functions:
  - name: pgBackup
    inputs:
      database: ${databases}
      provider: ${provider}
      hostname: ${hostname}
      repoDir: ${prefix}
      outputDir: ${outputDir}
      policy: ${retentionPolicyName}
      retentionValue: ${retentionPolicyValue}
      enableMetric: ${enableMetric}
      pushgatewayURL: ${pushgatewayURL}
      workloadKind: ${kind}
      workloadName: ${name}
      tempVolumeName: ${tmpVolumeName}
      storageSecretName: ${secretName}
  - name: stashPostBackup
    inputs:
      repoName: ${repoName}
      backupSession: ${backupSessionName}
      outputJsonDir: ${output-dir}
      outputVolumeName: ${output-volume-name}
```

#### pgRecovery

```yaml
# pgRecovery specifies required functions and their inputs to restore PostgreSQL database
apiVersion: stash.appscode.com/v1beta1
kind: Task
metadata:
  name: pgRecovery
spec:
  functions:
  - name: pgRecovery
    inputs:
      provider: ${provider}
      hostname: ${hostname}
      repoDir: ${prefix}
      outputDir: ${outputDir}
      enableMetric: ${enableMetric}
      pushgatewayURL: ${pushgatewayURL}
      workloadKind: ${kind}
      workloadName: ${name}
      tempVolumeName: ${tmpVolumeName}
      storageSecretName: ${secretName}
  - name: stashPostRecovery
    inputs:
      recoveryConfigurationName: ${recoveryConfigurationName}
      outputJsonDir: ${output-dir}
      outputVolumeName: ${output-volume-name}
```

#### clusterBackup

```yaml
# clusterBackup specifies required functions and their inputs to backup cluster yaml
apiVersion: stash.appscode.com/v1beta1
kind: Task
metadata:
  name: clusterBackup
spec:
  functions:
  - name: clusterBackup
    inputs:
      sanitize: ${sanitize}
      provider: ${provider}
      hostname: ${hostname}
      repoDir: ${prefix}
      outputDir: ${outputDir}
      policy: ${retentionPolicyName}
      retentionValue: ${retentionPolicyValue}
      enableMetric: ${enableMetric}
      pushgatewayURL: ${pushgatewayURL}
      workloadKind: ${kind}
      workloadName: ${name}
      tempVolumeName: ${tmpVolumeName}
      storageSecretName: ${secretName}
  - name: stashPostBackup
    inputs:
      repoName: ${repoName}
      backupSession: ${backupSessionName}
      outputJsonDir: ${output-dir}
      outputVolumeName: ${output-volume-name}
```
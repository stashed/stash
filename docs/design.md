# Design Discussion

Discuss about possible design and feature of stash.

## Features

Stash should be able to perform following task:

- **Volume Backup:**
  - [x] Backup Kubernetes Volumes.  - [x] Recover Backed up volumes.
  - [x] Schedule backup.
  - [ ] Instant backup.
  - [ ] Recover in same volume. Ref: [stash/issues/600](https://github.com/appscode/stash/issues/600)
  - [ ] Default backup. Ref:[stash/issues/367](https://github.com/appscode/stash/issues/367)
  - [ ] Auto recover if backup exist. [stash/issues/633](https://github.com/appscode/stash/issues/633)
- **Database Backup:**
  - [ ] Backup & Restore database deployed with KubeDB
  - [ ] Backup & Restore database deployed with other operators.
  - [ ] Schedule back-up.
  - [ ] Instant back-up.
- **Cluster YAML Backup:**
  - [ ] Backup Cluster resources YAML.
  - [ ] Restore Cluster from backed up YAML (advanced & priority low).

- **Monitoring:**
  Stash should export following monitoring metrics.
  - [ ] Total successful backup.
  - [ ] Total failed backup.
  - [ ] Last backup time.
  - [ ] Last backup duration(total of all file groups).
  - [ ] Total snapshot.
  - [ ] Recovery metrics.
- **Integration:**
  - [ ] Leverage AppsCode's native resources (AppBinding, AppRef, Vault etc.) while integrating with AppsCode cloud, KubeDB.
  - [ ] Provide simple API and examples to integrate with other Platforms.
- **Others:**
  - [ ] Stash should be able to fetch repository information from backend in case of cluster disaster where `Repository` crd is lost.
  - [ ] Pass additional env. or annotation to backup job/container. Ref: [stash/issues/621](https://github.com/appscode/stash/issues/621).

## Custom Resources Definition (CRD)

Custom Resources:

- Repository
- Restic
- Recovery
- BackupAgent/BackupTemplate

### Repository

Requirements:

- Should be created by user.
- `spec` section should contain backend information.
- `status` section should contain backup information in this Repository (`status` section should be handled by Stash itself).
- One Repository should be usable to backup multiple workloads.
- If a repository already exist in the backend, it should be synced with the existing one. (Maybe we can backup updated `Repository` yaml after each backup using `osm`).

Sample YAML:

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  name: stash-backup
  namespace: demo
spec:
  backend:
    gcs:
      bucket: stash-backup-repo
      prefix: demo
    storageSecretName: gcs-secret
status:
  backupRepos:
  - type: Volumes
    workloadKind: Deployment
    workloadName: stash-demo
    resticName: restic-volume-demo
    resticNamespace: default
    repoDir: /volumes/deployment/stash-demo
    backupCount: 7
    firstBackupTime: 2018-04-10T05:10:11Z
    lastBackupDuration: 3.026137088s
    lastBackupTime: 2018-04-10T05:16:12Z
  - type: Database
    workloadKind: StatefulSet
    workloadName: quick-elasticsearch
    resticName: restic-database-demo
    resticNamespace: default
    repoDir: /databases/statefulset/quick-elasticsearch-0
    backupCount: 2
    firstBackupTime: 2018-04-10T05:10:11Z
    lastBackupDuration: 3.026137088s
    lastBackupTime: 2018-04-10T05:16:12Z
  - type: Cluster
    resticName: restic-cluster-demo
    resticNamespace: default
    repoDir: /cluster/clustername
    backupCount: 1
    firstBackupTime: 2018-04-10T05:10:11Z
    lastBackupDuration: 3.026137088s
    lastBackupTime: 2018-04-10T05:16:12Z
```

### Restic

Requirements:

- Should be able to indicate target type (Volume, Database, Cluster).
- Should be able to indicate backup type (Instant, Scheduled)
- Should be able to indicate backup mode (online, offline)
- Should be able to use AppsCode's native resources (AppBinding/AppRef etc.)

**Sample Restic for Volume backup:**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: restic-volume-demo
  namespace: default
spec:
  targetType: Volume
  backupType: Scheduled
  schedule: '@every 1m'
  backupMode: Online
  backupAgent: stash # indicate which image to use for backup
  repository:
    name: stash-backup
    namespace: demo
  volumeSelector:
    selector:
      matchLabels:
        app: stash-demo
    targetVolumes:
    - name: vol-1
      mountPath: /source/data
      directories:
      - path: /source/data
        retentionPolicyName: 'keep-last-5'
    - name: vol-2
      mountPath: /source/data
      directories:
      - path: /source/data
        retentionPolicyName: 'keep-last-5'
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

**Sample Restic for Database backup(with AppRef):**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: restic-database-demo
  namespace: default
spec:
  targetType: Database
  backupType: Scheduled
  schedule: '@every 1m'
  backupMode: Online
  backupAgent: mysqldump # indicate which image to use for backup
  prefix: "databases" # prefix of repository directory in backend
  repository:
    name: stash-backup
    namespace: demo
  databaseSelector:
    AppRef:
      name: quick-mysql
      namespace: demo
    retentionPolicyName: 'keep-last-5'
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

**Sample Restic for Database backup(with ConnectionURL):**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: restic-database-demo
  namespace: default
spec:
  targetType: Database
  backupType: Scheduled
  schedule: '@every 1m'
  backupMode: Online
  backupAgent: mysqldump # indicate which image to use for backup
  prefix: "databases" # prefix of repository directory in backend
  repository:
    name: stash-backup
    namespace: demo
  databaseSelector:
    connectionURL:
      url: "https://quick-elasticsearch.demo.svc:9200"
      database: mydb # to backup all databases user may provide "all"
      credentials:
        secretName: mysql-secret
        namespace: demo
        userKey: ADMIN_USER
        passwordKey: ROOT_PASSWORD
    retentionPolicyName: 'keep-last-5'
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

**Sample Restic for Database backup (with DatabaseAccessor):**

Maybe we can use a single Restic to backup all similar type (i.e. `MySQL`) databases.

*DatabaseAccessors:*

```go
type DatabaseAccessors struct{
  AppRef *util.AppRef
  ConnectionURL *ConnectionURL
}

type ConnectionURL struct {
  URL string
  Credentials *Credentials
}

type Credentials struct {
  SecretName string
  Namespace string
  UserKey string
  PasswordKey string
}
```

*Restic:*

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: restic-database-demo
  namespace: default
spec:
  targetType: Database
  backupType: Scheduled
  schedule: '@every 1m'
  backupMode: Online
  backupAgent: mysqldump # indicate which image to use for backup
  prefix: "databases" # prefix of repository directory in backend
  repository:
    name: stash-backup
    namespace: demo
  databaseSelector: # array of DatabaseAccessor
    databaseAccessors:
    - AppRef:
        name: quick-mysql
        namespace: demo
    - connectionURL:
        url: "https://quick-elasticsearch.demo.svc:9200"
        database: mydb # to backup all databases user may provide "all"
        credentials:
          secretName: mysql-secret
          namespace: demo
          userKey: ADMIN_USER
          passwordKey: ROOT_PASSWORD
    retentionPolicyName: 'keep-last-5'
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

**Sample Restic for Cluster backup:**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: restic-cluster-demo
  namespace: default
spec:
  targetType: Cluster
  backupType: Scheduled
  schedule: '@every 1m'
  backupAgent: osm # indicate which image to use for backup
  prefix: "cluster" # prefix of repository directory in backend
  repository:
    name: stash-backup
    namespace: demo
  clusterResources:
    retentionPolicyName: 'keep-last-5'
    sanitize: true
    parentsOnly: true # only backup the parent resources(i.e. Deployment) not childs (i.e. ReplicaSet, Pod) who has ownerRef set.
    resourceSelector:
    - group: ""
      resources:
      - pods
      - secrets
      - persistentvolumelaims
    - group: "apps/v1"
      resources:
      - deployments
      - daemonsets
      - replicasets
  retentionPolicies:
  - name: 'keep-last-5'
    keepLast: 5
    prune: true
```

### Recovery

Requirements:

- Should not depend on `Restic`.
- Should depend on `Repository` only for backend information.

**Sample Recovery crd for Volume recovery:**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: recovery-volume-demo
  namespace: default
spec:
  targetType: Volume
  recoveryAgent: stash
  repository:
    name: stash-backup
    namespace: default
  volumeRestorer:
    repoDir: /volumes/deployment/stash-demo
    snapshot: deployment.stash-demo-e0e9c272
    paths:
    - /source/data
    recoverTo:
    - mountPath: /source/data
      hostPath:
        path: /data/stash-test/restic-restored
```

**Sample Recovery crd for Database recovery(with AppRef):**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: recovery-database-demo
  namespace: default
spec:
  targetType: Database
  recoveryAgent: mysqldump
  repository:
    name: stash-backup
    namespace: default
  databaseRestorer:
    repoDir: /databases/statefulset/quick-elasticsearch-0
    snapshot: snapshot.name
    recoverTo:
      AppRef:
        name: quick-mysql
        namespace: demo
```

**Sample Recovery crd for Database recovery(with connectionURL):**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Recovery
metadata:
  name: recovery-database-demo
  namespace: default
spec:
  targetType: Database
  recoveryAgent: mysqldump
  repository:
    name: stash-backup
    namespace: default
  databaseRestorer:
    repoDir: /databases/statefulset/quick-elasticsearch-0
    snapshot: snapshot.name
    recoverTo:
      connectionURL:
        url: "https://quick-elasticsearch.demo.svc:9200"
        credentials:
          secretName: mysql-secret
          namespace: demo
          userKey: ADMIN_USER
          passwordKey: ROOT_PASSWORD
```

### BackupAgent / BackupTemplate / StashAgent / StashTools

Requirements:

- Specify the image and command to use for backup & recovery
- Should be created automatically when install Stash
- Should be non-namespaced crd
- User should be able to customize to use their own image or commands.

**Sample BackupAgent crd for Volume backup/recovery:**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: BackupAgent
metadata:
  name: stash
spec:
  image: appscode/stash:0.8.1
  commands:
    backup:
    - backup
    - --restic-name=restic-volume-demo
    - --workload-kind=Deployment
    - --workload-name=stash-demo
    - --run-via-cron=true
    - --pushgateway-url=http://stash-operator.kube-system.svc:56789
    - --enable-status-subresource=true
    - --use-kubeapiserver-fqdn-for-aks=true
    - --enable-analytics=true
    - --enable-rbac=true
    - --logtostderr=true
    - --alsologtostderr=false
    - --v=3
    - --stderrthreshold=0
    recovery:
    - recover
    - --recovery-name=recovery-volume-demo
    - --pushgateway-url=http://stash-operator.kube-system.svc:56789
    - --enable-status-subresource=true
    - --use-kubeapiserver-fqdn-for-aks=true
    - --enable-analytics=true
    - --enable-rbac=true
    - --logtostderr=true
    - --alsologtostderr=false
    - --v=3
    - --stderrthreshold=0
```

**Sample BackupAgent for Database backup/recovery:**

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: BackupAgent
metadata:
  name: mysqldump
spec:
  image: kubedb/mysql-tools:9.0
  commands:
    backup:
    - mysqldump -u ${DB_USER} --password=${DB_PASSWORD} -h ${DB_HOST} "$@" | restic -r {{RepoName}} backup
    recovery:
    - restic -r {{RepoName}} restore | mysql -u "$DB_USER" --password=${DB_PASSWORD} -h "$DB_HOST" "$@"
```

## Problems

**Backward compatibility:**

- We must support repository created by earlier version of Stash.
- We must support recovery a volume that backed up by earlier version of Stash.
- We must provide automatic migration facility while upgrading stash from earlier version.
- Backup taken by KubeDB is not compatible with Stash.

**Others:**

- Need more research on cluster backup & recovery.
---
title: Stash Cli
menu:
  product_stash_0.8.3:
    identifier: stash-cli
    name: Stash Cli
    parent: reference
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: reference
---
## stash cli

Stash CLI

### Synopsis

Kubectl plugin for Stash

```
stash cli [flags]
```

### Options

```
  -h, --help   help for cli
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --enable-status-subresource        If true, uses sub resource for crds.
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --service-name string              Stash service name. (default "stash-operator")
      --stderrthreshold severity         logs at or above this threshold go to stderr
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [stash](/docs/reference/stash.md)	 - Stash by AppsCode - Backup your Kubernetes Volumes
* [stash cli backup-pv](/docs/reference/stash_cli_backup-pv.md)	 - Backup persistent volume
* [stash cli copy-repository](/docs/reference/stash_cli_copy-repository.md)	 - Copy Repository and Secret
* [stash cli delete-snapshot](/docs/reference/stash_cli_delete-snapshot.md)	 - Delete a snapshot from repository backend
* [stash cli download](/docs/reference/stash_cli_download.md)	 - Download snapshots
* [stash cli trigger-backup](/docs/reference/stash_cli_trigger-backup.md)	 - Trigger a backup
* [stash cli unlock-local-repository](/docs/reference/stash_cli_unlock-local-repository.md)	 - Unlock Restic Repository with Local Backend
* [stash cli unlock-repository](/docs/reference/stash_cli_unlock-repository.md)	 - Unlock Restic Repository


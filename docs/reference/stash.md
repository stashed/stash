---
title: Stash
menu:
  product_stash_0.8.2:
    identifier: stash
    name: Stash
    parent: reference
    weight: 0

product_name: stash
menu_name: product_stash_0.8.2
section_menu_id: reference
aliases:
  - products/stash/0.8.2/reference/

---
## stash

Stash by AppsCode - Backup your Kubernetes Volumes

### Synopsis

Stash is a Kubernetes operator for restic. For more information, visit here: https://appscode.com/products/stash

### Options

```
      --alsologtostderr                  log to standard error as well as files
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --enable-status-subresource        If true, uses sub resource for crds.
  -h, --help                             help for stash
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

* [stash backup](/docs/reference/stash_backup.md)	 - Run Stash Backup
* [stash check](/docs/reference/stash_check.md)	 - Check restic backup
* [stash forget](/docs/reference/stash_forget.md)	 - Delete snapshots from a restic repository
* [stash recover](/docs/reference/stash_recover.md)	 - Recover restic backup
* [stash run](/docs/reference/stash_run.md)	 - Launch Stash Controller
* [stash scaledown](/docs/reference/stash_scaledown.md)	 - Scale down workload
* [stash snapshots](/docs/reference/stash_snapshots.md)	 - Get snapshots of restic repo
* [stash version](/docs/reference/stash_version.md)	 - Prints binary version number.


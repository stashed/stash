---
title: Stash Backup-Pg
menu:
  product_stash_0.8.3:
    identifier: stash-backup-pg
    name: Stash Backup-Pg
    parent: reference
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: reference
---
## stash backup-pg

Takes a backup of Postgres DB

### Synopsis

Takes a backup of Postgres DB

```
stash backup-pg [flags]
```

### Options

```
      --app-binding string               Name of the app binding
      --bucket string                    Name of the cloud bucket/container (keep empty for local backend)
      --enable-cache                     Specify whether to enable caching for restic
      --endpoint string                  Endpoint for s3/s3 compatible backend
  -h, --help                             help for backup-pg
      --hostname string                  Name of the host machine (default "host-0")
      --kubeconfig string                Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                    The address of the Kubernetes API server (overrides any value in kubeconfig)
      --max-connections int              Specify maximum concurrent connections for GCS, Azure and B2 backend
      --metrics-dir string               Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)
      --metrics-enabled                  Specify whether to export Prometheus metrics
      --metrics-labels strings           Labels to apply in exported metrics
      --metrics-pushgateway-url string   Pushgateway URL where the metrics will be pushed
      --namespace string                 Namespace of Backup/Restore Session (default "default")
      --output-dir string                Directory where output.json file will be written (keep empty if you don't need to write output in file)
      --path string                      Directory inside the bucket where backup will be stored
      --pg-args string                   Additional arguments
      --provider string                  Backend provider (i.e. gcs, s3, azure etc)
      --rest-server-url string           URL for rest backend
      --retention-dry-run                Specify whether to test retention policy without deleting actual data
      --retention-keep-daily int         Specify value for retention strategy
      --retention-keep-hourly int        Specify value for retention strategy
      --retention-keep-last int          Specify value for retention strategy
      --retention-keep-monthly int       Specify value for retention strategy
      --retention-keep-tags strings      Specify value for retention strategy
      --retention-keep-weekly int        Specify value for retention strategy
      --retention-keep-yearly int        Specify value for retention strategy
      --retention-prune                  Specify whether to prune old snapshot data
      --scratch-dir string               Temporary directory (default "/tmp")
      --secret-dir string                Directory where storage secret has been mounted
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


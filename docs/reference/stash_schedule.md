---
title: Stash Schedule
menu:
  product_stash_0.5.1:
    identifier: stash-schedule
    name: Stash Schedule
    parent: reference
product_name: stash
left_menu: product_stash_0.5.1
section_menu_id: reference
---
## stash schedule

Run Stash cron daemon

### Synopsis


Run Stash cron daemon

```
stash schedule [flags]
```

### Options

```
  -h, --help                     help for schedule
      --kubeconfig string        Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string            The address of the Kubernetes API server (overrides any value in kubeconfig)
      --pushgateway-url string   URL of Prometheus pushgateway used to cache backup metrics (default "http://stash-operator.kube-system.svc:56789")
      --restic-name string       Name of the Restic used as configuration.
      --resync-period duration   If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 5m0s)
      --scratch-dir emptyDir     Directory used to store temporary files. Use an emptyDir in Kubernetes. (default "/tmp")
      --workload-kind string     Kind of workload where sidecar pod is added.
      --workload-name string     Name of workload where sidecar pod is added.
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Analytics (default true)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [stash](/docs/reference/stash.md)	 - Stash by AppsCode - Backup your Kubernetes Volumes


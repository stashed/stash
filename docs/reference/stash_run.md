---
title: Stash Run
menu:
  product_stash_0.6.2:
    identifier: stash-run
    name: Stash Run
    parent: reference
product_name: stash
menu_name: product_stash_0.6.2
section_menu_id: reference
---
## stash run

Run Stash operator

### Synopsis

Run Stash operator

```
stash run [flags]
```

### Options

```
      --address string           Address to listen on for web interface and telemetry. (default ":56790")
  -h, --help                     help for run
      --kubeconfig string        Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string            The address of the Kubernetes API server (overrides any value in kubeconfig)
      --rbac                     Enable RBAC for operator
      --resync-period duration   If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 5m0s)
      --scratch-dir emptyDir     Directory used to store temporary files. Use an emptyDir in Kubernetes. (default "/tmp")
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Analytics (default true)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [stash](/docs/reference/stash.md)	 - Stash by AppsCode - Backup your Kubernetes Volumes


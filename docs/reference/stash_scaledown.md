---
title: Stash Scaledown
menu:
  product_stash_0.7.0-rc.4:
    identifier: stash-scaledown
    name: Stash Scaledown
    parent: reference
product_name: stash
menu_name: product_stash_0.7.0-rc.4
section_menu_id: reference
---
## stash scaledown

Scale down workload

### Synopsis

Scale down workload

```
stash scaledown [flags]
```

### Options

```
  -h, --help                help for scaledown
      --kubeconfig string   Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string       The address of the Kubernetes API server (overrides any value in kubeconfig)
      --selector string     Label used to select Restic's workload
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [stash](/docs/reference/stash.md)	 - Stash by AppsCode - Backup your Kubernetes Volumes


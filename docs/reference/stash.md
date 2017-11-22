---
title: Reference | Stash
description: Stash Reference
menu:
  product_stash_0.5.1:
    identifier: reference-stash
    name: Stash
    parent: reference
    weight: 10
product_name: stash
left_menu: product_stash_0.5.1
section_menu_id: reference
aliases:
  - products/stash/0.5.1/reference/
---
## stash

Stash by AppsCode - Backup your Kubernetes Volumes

### Synopsis


Stash is a Kubernetes operator for restic. For more information, visit here: https://github.com/appscode/stash/tree/master/docs

### Options

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Analytics (default true)
  -h, --help                             help for stash
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [stash recover](stash_recover.md)	 - Recover restic backup
* [stash run](stash_run.md)	 - Run Stash operator
* [stash schedule](stash_schedule.md)	 - Run Stash cron daemon
* [stash version](stash_version.md)	 - Prints binary version number.


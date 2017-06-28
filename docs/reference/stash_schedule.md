## stash schedule

Run Stash cron daemon

### Synopsis


Run Stash cron daemon

```
stash schedule [flags]
```

### Options

```
      --analytics                Send analytical events to Google Analytics (default true)
      --app string               Name of app where sidecar pod is added
  -h, --help                     help for schedule
      --kubeconfig string        Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string            The address of the Kubernetes API server (overrides any value in kubeconfig)
      --name string              Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --namespace string         The address of the Kubernetes API server (overrides any value in kubeconfig)
      --pod-labels-path string   Path to pod labels file mounted via Kubernetes Downward api (default "/etc/labels")
      --prefix-hostname          If set, adds Hostname as prefix to repository. This should be true for StatefulSets & DaemonSets. This should be false in all other cases. (default true)
      --pushgateway-url string   URL of Prometheus pushgateway used to cache backup metrics (default "http://stash-operator.kube-system.svc:56789")
      --scratch-dir emptyDir     Directory used to store temporary files. Use an emptyDir in Kubernetes. (default "/tmp")
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [stash](stash.md)	 - Stash by AppsCode - Backup your Kubernetes Volumes



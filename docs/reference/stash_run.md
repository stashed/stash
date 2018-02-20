---
title: Stash Run
menu:
  product_stash_0.7.0-rc.0:
    identifier: stash-run
    name: Stash Run
    parent: reference
product_name: stash
menu_name: product_stash_0.7.0-rc.0
section_menu_id: reference
---
## stash run

Launch Stash Controller

### Synopsis

Launch Stash Controller

```
stash run [flags]
```

### Options

```
      --audit-log-format string                                 Format of saved audits. "legacy" indicates 1-line text format for each event. "json" indicates structured json format. Requires the 'AdvancedAuditing' feature gate. Known formats are legacy,json. (default "json")
      --audit-log-maxage int                                    The maximum number of days to retain old audit log files based on the timestamp encoded in their filename.
      --audit-log-maxbackup int                                 The maximum number of old audit log files to retain.
      --audit-log-maxsize int                                   The maximum size in megabytes of the audit log file before it gets rotated.
      --audit-log-path string                                   If set, all requests coming to the apiserver will be logged to this file.  '-' means standard out.
      --audit-policy-file string                                Path to the file that defines the audit policy configuration. Requires the 'AdvancedAuditing' feature gate. With AdvancedAuditing, a profile is required to enable auditing.
      --audit-webhook-batch-buffer-size int                     The size of the buffer to store events before batching and sending to the webhook. Only used in batch mode. (default 10000)
      --audit-webhook-batch-initial-backoff duration            The amount of time to wait before retrying the first failed requests. Only used in batch mode. (default 10s)
      --audit-webhook-batch-max-size int                        The maximum size of a batch sent to the webhook. Only used in batch mode. (default 400)
      --audit-webhook-batch-max-wait duration                   The amount of time to wait before force sending the batch that hadn't reached the max size. Only used in batch mode. (default 30s)
      --audit-webhook-batch-throttle-burst int                  Maximum number of requests sent at the same moment if ThrottleQPS was not utilized before. Only used in batch mode. (default 15)
      --audit-webhook-batch-throttle-qps float32                Maximum average number of requests per second. Only used in batch mode. (default 10)
      --audit-webhook-config-file string                        Path to a kubeconfig formatted file that defines the audit webhook configuration. Requires the 'AdvancedAuditing' feature gate.
      --audit-webhook-mode string                               Strategy for sending audit events. Blocking indicates sending events should block server responses. Batch causes the webhook to buffer and send events asynchronously. Known modes are batch,blocking. (default "batch")
      --authentication-kubeconfig string                        kubeconfig file pointing at the 'core' kubernetes server with enough rights to create tokenaccessreviews.authentication.k8s.io.
      --authentication-skip-lookup                              If false, the authentication-kubeconfig will be used to lookup missing authentication configuration from the cluster.
      --authentication-token-webhook-cache-ttl duration         The duration to cache responses from the webhook token authenticator. (default 10s)
      --authorization-kubeconfig string                         kubeconfig file pointing at the 'core' kubernetes server with enough rights to create  subjectaccessreviews.authorization.k8s.io.
      --authorization-webhook-cache-authorized-ttl duration     The duration to cache 'authorized' responses from the webhook authorizer. (default 10s)
      --authorization-webhook-cache-unauthorized-ttl duration   The duration to cache 'unauthorized' responses from the webhook authorizer. (default 10s)
      --bind-address ip                                         The IP address on which to listen for the --secure-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank, all interfaces will be used (0.0.0.0). (default 0.0.0.0)
      --cert-dir string                                         The directory where the TLS certs are located. If --tls-cert-file and --tls-private-key-file are provided, this flag will be ignored. (default "apiserver.local.config/certificates")
      --client-ca-file string                                   If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate.
      --contention-profiling                                    Enable lock contention profiling, if profiling is enabled
      --docker-registry string                                  Docker image registry for sidecar, init-container, check-job, recovery-job and kubectl-job (default "appscode")
      --enable-swagger-ui                                       Enables swagger ui on the apiserver at /swagger-ui
  -h, --help                                                    help for run
      --image-tag string                                        Image tag for sidecar, init-container, check-job and recovery-job (default "canary")
      --kubeconfig string                                       kubeconfig file pointing at the 'core' kubernetes server.
      --ops-address string                                      Address to listen on for web interface and telemetry. (default ":56790")
      --profiling                                               Enable profiling via web interface host:port/debug/pprof/ (default true)
      --rbac                                                    Enable RBAC for operator
      --requestheader-allowed-names strings                     List of client certificate common names to allow to provide usernames in headers specified by --requestheader-username-headers. If empty, any client certificate validated by the authorities in --requestheader-client-ca-file is allowed.
      --requestheader-client-ca-file string                     Root certificate bundle to use to verify client certificates on incoming requests before trusting usernames in headers specified by --requestheader-username-headers
      --requestheader-extra-headers-prefix strings              List of request header prefixes to inspect. X-Remote-Extra- is suggested. (default [x-remote-extra-])
      --requestheader-group-headers strings                     List of request headers to inspect for groups. X-Remote-Group is suggested. (default [x-remote-group])
      --requestheader-username-headers strings                  List of request headers to inspect for usernames. X-Remote-User is common. (default [x-remote-user])
      --resync-period duration                                  If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 10m0s)
      --scratch-dir emptyDir                                    Directory used to store temporary files. Use an emptyDir in Kubernetes. (default "/tmp")
      --secure-port int                                         The port on which to serve HTTPS with authentication and authorization. If 0, don't serve HTTPS at all. (default 443)
      --tls-ca-file string                                      If set, this certificate authority will used for secure access from Admission Controllers. This must be a valid PEM-encoded CA bundle. Altneratively, the certificate authority can be appended to the certificate provided by --tls-cert-file.
      --tls-cert-file string                                    File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert). If HTTPS serving is enabled, and --tls-cert-file and --tls-private-key-file are not provided, a self-signed certificate and key are generated for the public address and saved to the directory specified by --cert-dir.
      --tls-private-key-file string                             File containing the default x509 private key matching --tls-cert-file.
      --tls-sni-cert-key namedCertKey                           A pair of x509 certificate and private key file paths, optionally suffixed with a list of domain patterns which are fully qualified domain names, possibly with prefixed wildcard segments. If no domain patterns are provided, the names of the certificate are extracted. Non-wildcard matches trump over wildcard matches, explicit domain patterns trump over extracted names. For multiple key/certificate pairs, use the --tls-sni-cert-key multiple times. Examples: "example.crt,example.key" or "foo.crt,foo.key:*.foo.com,foo.com". (default [])
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


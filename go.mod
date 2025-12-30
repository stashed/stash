module stash.appscode.dev/stash

go 1.25

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/gogo/protobuf v1.3.2
	github.com/kubernetes-csi/external-snapshotter/client/v8 v8.4.0
	github.com/onsi/ginkgo/v2 v2.22.1
	github.com/onsi/gomega v1.36.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.10.1
	github.com/spf13/pflag v1.0.9
	go.bytebuilders.dev/audit v0.0.46
	go.bytebuilders.dev/license-proxyserver v0.0.24
	go.bytebuilders.dev/license-verifier v0.14.10
	go.bytebuilders.dev/license-verifier/kubernetes v0.14.10
	golang.org/x/text v0.32.0
	gomodules.xyz/blobfs v0.2.2
	gomodules.xyz/cert v1.6.0
	gomodules.xyz/encoding v0.0.8
	gomodules.xyz/envsubst v0.2.0
	gomodules.xyz/flags v0.1.3
	gomodules.xyz/go-sh v0.1.0
	gomodules.xyz/logs v0.0.7
	gomodules.xyz/pointer v0.1.0
	gomodules.xyz/runtime v0.3.0
	gomodules.xyz/stow v0.2.4
	gomodules.xyz/x v0.0.17
	k8s.io/api v0.34.3
	k8s.io/apiextensions-apiserver v0.34.3
	k8s.io/apimachinery v0.34.3
	k8s.io/apiserver v0.34.3
	k8s.io/client-go v0.34.3
	k8s.io/component-base v0.34.3
	k8s.io/klog/v2 v2.130.1
	k8s.io/kube-aggregator v0.34.3
	k8s.io/kubernetes v1.34.3
	kmodules.xyz/client-go v0.34.2
	kmodules.xyz/constants v0.0.0-20250815043538-9de88de78858
	kmodules.xyz/csi-utils v0.34.0
	kmodules.xyz/custom-resources v0.34.0
	kmodules.xyz/objectstore-api v0.34.0
	kmodules.xyz/offshoot-api v0.34.0
	kmodules.xyz/openshift v0.29.0
	kmodules.xyz/prober v0.34.0
	kmodules.xyz/webhook-runtime v0.34.0
	sigs.k8s.io/controller-runtime v0.22.4
	stash.appscode.dev/apimachinery v0.42.2-0.20251230090158-1034b727fe48
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/auth v0.15.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.30 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/PuerkitoBio/purell v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.6 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.17 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.21 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.69 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.78.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.39.1 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2 // indirect
	github.com/codegangsta/inject v0.0.0-20150114235600-33e0aa1cb7c0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.18.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.0.3+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.4 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.11 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.2 // indirect
	github.com/go-openapi/swag v0.25.1 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.1 // indirect
	github.com/go-openapi/swag/conv v0.25.1 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.1 // indirect
	github.com/go-openapi/swag/loading v0.25.1 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/netutils v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/cel-go v0.26.0 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-containerregistry v0.20.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.1-0.20220621161143-b0104c826a24 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nats-io/nats.go v1.47.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ncw/swift v1.0.49 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rancher/norman v0.5.2 // indirect
	github.com/rancher/rancher/pkg/client v0.0.0-20250220153925-3abb578f42fe // indirect
	github.com/rancher/wrangler/v3 v3.2.0-rc.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/vbatts/tar-split v0.12.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.etcd.io/etcd/api/v3 v3.6.4 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.4 // indirect
	go.etcd.io/etcd/client/v3 v3.6.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/sdk v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	gocloud.dev v0.41.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gomodules.xyz/clock v0.0.0-20200817085942-06523dba733f // indirect
	gomodules.xyz/counter v0.0.1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	gomodules.xyz/jsonpath v0.0.2 // indirect
	gomodules.xyz/mergo v0.3.13 // indirect
	gomodules.xyz/password-generator v0.2.9 // indirect
	gomodules.xyz/sets v0.2.1 // indirect
	gomodules.xyz/sync v0.1.0 // indirect
	gomodules.xyz/wait v0.2.0 // indirect
	google.golang.org/api v0.228.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/grpc v1.72.1 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.0 // indirect
	k8s.io/cli-runtime v0.34.3 // indirect
	k8s.io/component-helpers v0.34.3 // indirect
	k8s.io/controller-manager v0.34.3 // indirect
	k8s.io/kms v0.34.3 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4 // indirect
	kmodules.xyz/apiversion v0.2.0 // indirect
	kmodules.xyz/go-containerregistry v0.0.15 // indirect
	kmodules.xyz/resource-metadata v0.40.2 // indirect
	kmodules.xyz/resource-metrics v0.34.0 // indirect
	moul.io/http2curl/v2 v2.3.1-0.20221024080105-10c404f653f7 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.31.2 // indirect
	sigs.k8s.io/cli-utils v0.37.2 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/kustomize/api v0.20.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.20.1 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
	x-helm.dev/apimachinery v0.0.18 // indirect
)

replace github.com/Masterminds/sprig/v3 => github.com/gomodules/sprig/v3 v3.2.3-0.20220405051441-0a8a99bac1b8

replace sigs.k8s.io/controller-runtime => github.com/kmodules/controller-runtime v0.22.5-0.20251227114913-f011264689cd

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.6

replace k8s.io/apiserver => github.com/kmodules/apiserver v0.34.4-0.20251227112449-07fa35efc6fc

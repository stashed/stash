module stash.appscode.dev/stash

go 1.12

require (
	github.com/appscode/go v0.0.0-20191119085241-0887d8ec2ecc
	github.com/armon/circbuf v0.0.0-20150827004946-bbbad097214e
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/codeskyblue/go-sh v0.0.0-20190412065543-76bd3d59ff27
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/go-openapi/spec v0.19.3
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/gofuzz v1.0.0
	github.com/json-iterator/go v1.1.8
	github.com/kubernetes-csi/external-snapshotter v1.2.1-0.20191220025132-bbbadbc41e30
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/robfig/cron/v3 v3.0.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	gomodules.xyz/cert v1.0.2
	gomodules.xyz/envsubst v0.1.0
	gomodules.xyz/stow v0.2.3
	gopkg.in/ini.v1 v1.51.0
	k8s.io/api v0.0.0-20191122220107-b5267f2975e0
	k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery v0.16.5-beta.1
	k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kubernetes v1.16.3
	kmodules.xyz/client-go v0.0.0-20200116162153-e083ae16abca
	kmodules.xyz/constants v0.0.0-20191024095500-cd4313df4aa6
	kmodules.xyz/crd-schema-fuzz v0.0.0-20191129174258-81f984340891
	kmodules.xyz/custom-resources v0.0.0-20191130062942-f41b54f62419
	kmodules.xyz/objectstore-api v0.0.0-20191127144749-5881939b57f0
	kmodules.xyz/offshoot-api v0.0.0-20200103145223-2c4f520520d2
	kmodules.xyz/openshift v0.0.0-20191127145035-f6c48a90dbb7
	kmodules.xyz/prober v0.0.0-20191216013129-7c91e2b0edb2
	kmodules.xyz/webhook-runtime v0.0.0-20191127075323-d4bfdee6974d
	sigs.k8s.io/yaml v1.1.0
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.38.0
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/go-ansiterm => github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.0.0+incompatible
	github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.5.0
	github.com/Azure/go-autorest/autorest/azure/auth => github.com/Azure/go-autorest/autorest/azure/auth v0.2.0
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.1.0
	github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.2.0
	github.com/Azure/go-autorest/autorest/to => github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.1.0
	github.com/Azure/go-autorest/logger => github.com/Azure/go-autorest/logger v0.1.0
	github.com/Azure/go-autorest/tracing => github.com/Azure/go-autorest/tracing v0.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	google.golang.org/api => google.golang.org/api v0.6.1-0.20190607001116-5213b8090861
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => github.com/kmodules/apimachinery v0.0.0-20191119091232-0553326db082
	k8s.io/apiserver => github.com/kmodules/apiserver v0.0.0-20191119111000-36ac3646ae82
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubernetes => github.com/kmodules/kubernetes v1.17.0-alpha.0.0.20191127022853-9d027e3886fd
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b
	k8s.io/repo-infra => k8s.io/repo-infra v0.0.0-20181204233714-00fe14e3d1a3
	k8s.io/utils => k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	sigs.k8s.io/kustomize => sigs.k8s.io/kustomize v2.0.3+incompatible
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190817042607-6149e4549fca
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.1.0
)

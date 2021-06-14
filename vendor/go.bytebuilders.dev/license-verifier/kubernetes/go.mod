module go.bytebuilders.dev/license-verifier/kubernetes

go 1.16

require (
	github.com/gogo/protobuf v1.3.2
	go.bytebuilders.dev/license-verifier v0.9.2
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/kube-aggregator v0.21.0
	kmodules.xyz/client-go v0.0.0-20210514054158-27e164b43474
)

replace go.bytebuilders.dev/license-verifier => ./..

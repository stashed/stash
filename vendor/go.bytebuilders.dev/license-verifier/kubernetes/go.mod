module go.bytebuilders.dev/license-verifier/kubernetes

go 1.15

require (
	github.com/gogo/protobuf v1.3.2
	go.bytebuilders.dev/license-verifier v0.9.1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/kube-aggregator v0.21.0
	kmodules.xyz/client-go v0.0.0-20210504004915-de8d9776f2a1
)

replace go.bytebuilders.dev/license-verifier => ./..

replace cloud.google.com/go => cloud.google.com/go v0.54.0

replace github.com/golang/protobuf => github.com/golang/protobuf v1.4.3

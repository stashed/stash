module go.bytebuilders.dev/license-verifier/kubernetes

go 1.14

require (
	github.com/gogo/protobuf v1.3.1
	go.bytebuilders.dev/license-verifier v0.5.1
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/apiserver v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.18.9
	kmodules.xyz/client-go v0.0.0-20201105071625-0b277310b9b8
)

replace cloud.google.com/go => cloud.google.com/go v0.38.0

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

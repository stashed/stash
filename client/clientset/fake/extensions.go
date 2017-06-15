package fake

import (
	"github.com/appscode/restik/client/clientset"
apiv1 "k8s.io/client-go/pkg/api/v1"
"k8s.io/client-go/pkg/api"
"k8s.io/client-go/rest"
"k8s.io/client-go/testing"
"k8s.io/apimachinery/pkg/runtime"
"k8s.io/apimachinery/pkg/watch"
)

type FakeExtensionClient struct {
	*testing.Fake
}

var _ clientset.ExtensionInterface = &FakeExtensionClient{}

func NewFakeRestikClient(objects ...runtime.Object) *FakeExtensionClient {
	o := testing.NewObjectTracker(apiv1.Scheme, apiv1.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == "backup.appscode.com" {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, apiv1.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &FakeExtensionClient{&fakePtr}
}

func (m *FakeExtensionClient) Restiks(ns string) clientset.RestikInterface {
	return &FakeRestik{m.Fake, ns}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensionClient) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

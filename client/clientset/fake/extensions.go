package fake

import (
	"github.com/appscode/restik/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeRestikClient struct {
	*testing.Fake
}

var _ clientset.AppsCodeRestikInterface = &FakeRestikClient{}

func NewFakeRestikClient(objects ...runtime.Object) *FakeRestikClient {
	o := testing.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == "backup.appscode.com" {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &FakeRestikClient{&fakePtr}
}


func (m *FakeRestikClient) Restiks(ns string) clientset.RestikInterface {
	return &FakeRestik{m.Fake, ns}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeRestikClient) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

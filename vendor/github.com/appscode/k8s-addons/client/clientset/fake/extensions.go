package fake

import (
	"github.com/appscode/k8s-addons/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeExtensionClient struct {
	*testing.Fake
}

var _ clientset.AppsCodeExtensionInterface = &FakeExtensionClient{}

func NewFakeExtensionClient(objects ...runtime.Object) *FakeExtensionClient {
	o := testing.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == "appscode.com" {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &FakeExtensionClient{&fakePtr}
}

func (a *FakeExtensionClient) Ingress(namespace string) clientset.IngressInterface {
	return &FakeIngress{a.Fake, namespace}
}

func (a *FakeExtensionClient) Alert(namespace string) clientset.AlertInterface {
	return &FakeAlert{a.Fake, namespace}
}

func (m *FakeExtensionClient) Certificate(ns string) clientset.CertificateInterface {
	return &FakeCertificate{m.Fake, ns}
}

func (m *FakeExtensionClient) Backups(ns string) clientset.BackupInterface {
	return &FakeBackup{m.Fake, ns}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensionClient) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

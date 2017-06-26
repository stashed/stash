package framework

import (
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (f *Framework) EventuallyTPR(name string) GomegaAsyncAssertion {
	return Eventually(func() *extensions.ThirdPartyResource {
		obj, err := f.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

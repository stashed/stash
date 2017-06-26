package framework

import (
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyTPR(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		_, err := f.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(name, metav1.GetOptions{})
		return err
	})
}

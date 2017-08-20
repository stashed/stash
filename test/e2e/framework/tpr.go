package framework

import (
	sapi "github.com/appscode/stash/api"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) EventuallyTPR(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		_, err := f.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get("restic."+sapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// TPR group registration has 10 sec delay inside Kuberneteas api server. So, needs the extra check.
		_, err = f.StashClient.Restics(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		return err
	})
}

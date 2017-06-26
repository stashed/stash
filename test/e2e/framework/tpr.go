package framework

import (

	apiv1 "k8s.io/client-go/pkg/api/v1"
	sapi "github.com/appscode/stash/api"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
)

func (f *Framework) EventuallyTPR(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		fmt.Println(">>>>>>>>>>>>>>>>>>", name)
		_, err := f.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get("restic." + sapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// TPR group registration has 10 sec delay inside Kuberneteas api server. So, needs the extra check.
		_, err = f.stashClient.Restics(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		fmt.Println(">>>>>>>>>>>>>>>>>>", err)
		return err
	})
}

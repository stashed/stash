package framework

import (
	"github.com/appscode/kutil"
	sapi "github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyCRD(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		return kutil.WaitForCRDReady(
			f.KubeClient.CoreV1().RESTClient(),
			[]*apiextensions.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: api.ResourceTypeRestic + "." + api.SchemeGroupVersion.Group,
					},
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:   sapi.GroupName,
						Version: api.SchemeGroupVersion.Version,
						Names: apiextensions.CustomResourceDefinitionNames{
							Plural: api.ResourceTypeRestic,
						},
					},
				},
			},
		)
	})
}

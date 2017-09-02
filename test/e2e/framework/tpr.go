package framework

import (
	sapi "github.com/appscode/stash/apis/stash"
	sapi_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	. "github.com/onsi/gomega"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyCRD(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		return util.WaitForCRDReady(
			f.KubeClient.CoreV1().RESTClient(),
			[]*apiextensions.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: sapi_v1alpha1.ResourceTypeRestic + "." + sapi_v1alpha1.SchemeGroupVersion.Group,
					},
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:   sapi.GroupName,
						Version: sapi_v1alpha1.SchemeGroupVersion.Version,
						Names: apiextensions.CustomResourceDefinitionNames{
							Plural: sapi_v1alpha1.ResourceTypeRestic,
						},
					},
				},
			},
		)
	})
}

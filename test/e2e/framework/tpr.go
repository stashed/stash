package framework

import (
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	sapi "github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyCRD(name string) GomegaAsyncAssertion {
	return Eventually(func() error {
		return apiext_util.WaitForCRDReady(
			f.KubeClient.CoreV1().RESTClient(),
			[]*apiext.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: api.ResourceTypeBackup + "." + api.SchemeGroupVersion.Group,
					},
					Spec: apiext.CustomResourceDefinitionSpec{
						Group:   sapi.GroupName,
						Version: api.SchemeGroupVersion.Version,
						Names: apiext.CustomResourceDefinitionNames{
							Plural: api.ResourceTypeBackup,
						},
					},
				},
			},
		)
	})
}

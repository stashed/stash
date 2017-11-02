package framework

import (
	"github.com/appscode/stash/pkg/util"
	"k8s.io/api/admissionregistration/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) InitializerForResources(resources []string) v1alpha1.InitializerConfiguration {
	return v1alpha1.InitializerConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "stash-initializer-config",
		},
		Initializers: []v1alpha1.Initializer{
			{
				Name: util.StashInitializerName,
				Rules: []v1alpha1.Rule{
					{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   resources,
					},
				},
			},
		},
	}
}

func (f *Framework) CreateInitializerConfiguration(config v1alpha1.InitializerConfiguration) error {
	_, err := f.KubeClient.AdmissionregistrationV1alpha1().InitializerConfigurations().Create(&config)
	return err
}

func (f *Framework) DeleteInitializerConfiguration(meta metav1.ObjectMeta) error {
	return f.KubeClient.AdmissionregistrationV1alpha1().InitializerConfigurations().Delete(meta.Name, nil)
}

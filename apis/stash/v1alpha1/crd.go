package v1alpha1

import (
	sapi "github.com/appscode/stash/apis/stash"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c Restic) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   sapi.ResourceTypeRestic + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "stash"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   sapi.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   sapi.ResourceNameRestic,
				Plural:     sapi.ResourceTypeRestic,
				Kind:       sapi.ResourceKindRestic,
				ShortNames: []string{"rst"},
			},
		},
	}
}

func (c Recovery) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   sapi.ResourceTypeRecovery + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "stash"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   sapi.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   sapi.ResourceNameRecovery,
				Plural:     sapi.ResourceTypeRecovery,
				Kind:       sapi.ResourceKindRecovery,
				ShortNames: []string{"rec"},
			},
		},
	}
}

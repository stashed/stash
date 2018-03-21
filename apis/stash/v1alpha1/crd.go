package v1alpha1

import (
	"github.com/appscode/stash/apis/stash"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c Restic) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ResourceTypeRestic + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "stash"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   stash.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   ResourceNameRestic,
				Plural:     ResourceTypeRestic,
				Kind:       ResourceKindRestic,
				ShortNames: []string{"rst"},
			},
		},
	}
}

func (c Recovery) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ResourceTypeRecovery + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "stash"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   stash.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   ResourceNameRecovery,
				Plural:     ResourceTypeRecovery,
				Kind:       ResourceKindRecovery,
				ShortNames: []string{"rec"},
			},
		},
	}
}

func (c Repository) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ResourceTypeRepository + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "stash"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   stash.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   ResourceNameRepository,
				Plural:     ResourceTypeRepository,
				Kind:       ResourceKindRepository,
				ShortNames: []string{"rec"},
			},
		},
	}
}

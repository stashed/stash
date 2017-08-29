package install

import (
	rapi "github.com/appscode/stash/api"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/pkg/api"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  rapi.GroupName,
			VersionPreferenceOrder:     []string{rapi.V1alpha1SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/appscode/stash/api",
			RootScopedKinds:            sets.NewString("CustomResourceDefinition"),
			AddInternalObjectsToScheme: rapi.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			rapi.V1alpha1SchemeGroupVersion.Version: rapi.V1alpha1AddToScheme,
		},
	).Announce(api.GroupFactoryRegistry).RegisterAndEnable(api.Registry, api.Scheme); err != nil {
		panic(err)
	}
}

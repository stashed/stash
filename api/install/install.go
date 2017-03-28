package install

import (
	aci "github.com/appscode/restik/api"
	"k8s.io/kubernetes/pkg/apimachinery/announced"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	"k8s.io/kubernetes/pkg/util/sets"
)

func init() {
	if !registered.IsRegistered(aci.GroupName) {
		if err := announced.NewGroupMetaFactory(
			&announced.GroupMetaFactoryArgs{
				GroupName:                  aci.GroupName,
				VersionPreferenceOrder:     []string{aci.V1beta1SchemeGroupVersion.Version},
				ImportPrefix:               "github.com/appscode/restik/api",
				RootScopedKinds:            sets.NewString("PodSecurityPolicy", "ThirdPartyResource"),
				AddInternalObjectsToScheme: aci.AddToScheme,
			},
			announced.VersionToSchemeFunc{
				aci.V1beta1SchemeGroupVersion.Version: aci.V1betaAddToScheme,
			},
		).Announce().RegisterAndEnable(); err != nil {
			panic(err)
		}
	}
}

package install

import (
	"github.com/appscode/restik/api"
	"k8s.io/kubernetes/pkg/apimachinery/announced"
	"k8s.io/kubernetes/pkg/util/sets"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  api.GroupName,
			VersionPreferenceOrder:     []string{api.V1beta1SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/appscode/restik/api",
			RootScopedKinds:            sets.NewString("PodSecurityPolicy", "ThirdPartyResource"),
			AddInternalObjectsToScheme: api.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			api.V1beta1SchemeGroupVersion.Version: api.V1betaAddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}

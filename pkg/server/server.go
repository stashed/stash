package server

import (
	"fmt"
	"strings"

	hookapi "github.com/appscode/stash/pkg/admission/api"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/registry/admissionreview"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apimachinery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	restclient "k8s.io/client-go/rest"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	admission.AddToScheme(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type StashConfig struct {
	GenericConfig    *genericapiserver.RecommendedConfig
	ControllerConfig *controller.ControllerConfig
}

// StashServer contains state for a Kubernetes cluster master/api server.
type StashServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	Controller       *controller.StashController
}

func (op *StashServer) Run(stopCh <-chan struct{}) error {
	go op.Controller.Run(stopCh)
	return op.GenericAPIServer.PrepareRun().Run(stopCh)
}

type completedConfig struct {
	GenericConfig    genericapiserver.CompletedConfig
	ControllerConfig *controller.ControllerConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *StashConfig) Complete() CompletedConfig {
	completedCfg := completedConfig{
		c.GenericConfig.Complete(),
		c.ControllerConfig,
	}

	completedCfg.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "1",
	}

	return CompletedConfig{&completedCfg}
}

// New returns a new instance of StashServer from the given config.
func (c completedConfig) New() (*StashServer, error) {
	genericServer, err := c.GenericConfig.New("stash-apiserver", genericapiserver.EmptyDelegate) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}
	ctrl, err := c.ControllerConfig.New()
	if err != nil {
		return nil, err
	}

	s := &StashServer{
		GenericAPIServer: genericServer,
		Controller:       ctrl,
	}

	inClusterConfig, err := restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}

	for _, versionMap := range admissionHooksByGroupThenVersion(c.ControllerConfig.AdmissionHooks...) {
		accessor := meta.NewAccessor()
		versionInterfaces := &meta.VersionInterfaces{
			ObjectConvertor:  Scheme,
			MetadataAccessor: accessor,
		}
		interfacesFor := func(version schema.GroupVersion) (*meta.VersionInterfaces, error) {
			if version != admission.SchemeGroupVersion {
				return nil, fmt.Errorf("unexpected version %v", version)
			}
			return versionInterfaces, nil
		}
		restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{admission.SchemeGroupVersion}, interfacesFor)
		// TODO we're going to need a later k8s.io/apiserver so that we can get discovery to list a different group version for
		// our endpoint which we'll use to back some custom storage which will consume the AdmissionReview type and give back the correct response
		apiGroupInfo := genericapiserver.APIGroupInfo{
			GroupMeta: apimachinery.GroupMeta{
				// filled in later
				//GroupVersion:  admissionVersion,
				//GroupVersions: []schema.GroupVersion{admissionVersion},

				SelfLinker:    runtime.SelfLinker(accessor),
				RESTMapper:    restMapper,
				InterfacesFor: interfacesFor,
				InterfacesByVersion: map[schema.GroupVersion]*meta.VersionInterfaces{
					admission.SchemeGroupVersion: versionInterfaces,
				},
			},
			VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
			// TODO unhardcode this.  It was hardcoded before, but we need to re-evaluate
			OptionsExternalVersion: &schema.GroupVersion{Version: "v1"},
			Scheme:                 Scheme,
			ParameterCodec:         metav1.ParameterCodec,
			NegotiatedSerializer:   Codecs,
		}

		for _, admissionHooks := range versionMap {
			for i := range admissionHooks {
				admissionHook := admissionHooks[i]
				admissionResource, singularResourceType := admissionHook.Resource()
				admissionVersion := admissionResource.GroupVersion()

				restMapper.AddSpecific(
					admission.SchemeGroupVersion.WithKind("AdmissionReview"),
					admissionResource,
					admissionVersion.WithResource(singularResourceType),
					meta.RESTScopeRoot)

				// just overwrite the groupversion with a random one.  We don't really care or know.
				apiGroupInfo.GroupMeta.GroupVersions = append(apiGroupInfo.GroupMeta.GroupVersions, admissionVersion)

				admissionReview := admissionreview.NewREST(admissionHook.Admit)
				v1alpha1storage := map[string]rest.Storage{
					admissionResource.Resource: admissionReview,
				}
				apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version] = v1alpha1storage
			}
		}

		// just prefer the first one in the list for consistency
		apiGroupInfo.GroupMeta.GroupVersion = apiGroupInfo.GroupMeta.GroupVersions[0]

		if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return nil, err
		}
	}

	for _, hook := range c.ControllerConfig.AdmissionHooks {
		postStartName := postStartHookName(hook)
		if len(postStartName) == 0 {
			continue
		}
		s.GenericAPIServer.AddPostStartHookOrDie(postStartName,
			func(context genericapiserver.PostStartHookContext) error {
				return hook.Initialize(inClusterConfig, context.StopCh)
			},
		)
	}

	return s, nil
}

func postStartHookName(hook hookapi.AdmissionHook) string {
	var ns []string
	gvr, _ := hook.Resource()
	ns = append(ns, fmt.Sprintf("admit-%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group))
	if len(ns) == 0 {
		return ""
	}
	return strings.Join(append(ns, "init"), "-")
}

func admissionHooksByGroupThenVersion(admissionHooks ...hookapi.AdmissionHook) map[string]map[string][]hookapi.AdmissionHook {
	ret := map[string]map[string][]hookapi.AdmissionHook{}

	for i := range admissionHooks {
		hook := admissionHooks[i]
		gvr, _ := hook.Resource()
		group, ok := ret[gvr.Group]
		if !ok {
			group = map[string][]hookapi.AdmissionHook{}
			ret[gvr.Group] = group
		}
		group[gvr.Version] = append(group[gvr.Version], hook)
	}
	return ret
}

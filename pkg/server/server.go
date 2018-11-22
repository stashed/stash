package server

import (
	"fmt"
	"os"
	"strings"

	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	admissionreview "github.com/appscode/kubernetes-webhook-util/registry/admissionreview/v1beta1"
	reg_util "github.com/appscode/kutil/admissionregistration/v1beta1"
	dynamic_util "github.com/appscode/kutil/dynamic"
	"github.com/appscode/stash/apis/repositories"
	"github.com/appscode/stash/apis/repositories/install"
	"github.com/appscode/stash/apis/repositories/v1alpha1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/eventer"
	snapregistry "github.com/appscode/stash/pkg/registry/snapshot"
	admission "k8s.io/api/admission/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	store "kmodules.xyz/objectstore-api/api/v1"
)

const (
	apiserviceName = "v1alpha1.admission.stash.appscode.com"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	install.Install(Scheme)
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
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   *controller.Config
}

// StashServer contains state for a Kubernetes cluster master/api server.
type StashServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	Controller       *controller.StashController
}

func (op *StashServer) Run(stopCh <-chan struct{}) error {
	// sync cache
	go op.Controller.Run(stopCh)
	return op.GenericAPIServer.PrepareRun().Run(stopCh)
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *controller.Config
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *StashConfig) Complete() CompletedConfig {
	completedCfg := completedConfig{
		c.GenericConfig.Complete(),
		c.ExtraConfig,
	}

	completedCfg.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "1",
	}

	return CompletedConfig{&completedCfg}
}

// New returns a new instance of StashServer from the given config.
func (c completedConfig) New() (*StashServer, error) {
	genericServer, err := c.GenericConfig.New("stash-apiserver", genericapiserver.NewEmptyDelegate()) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}
	ctrl, err := c.ExtraConfig.New()
	if err != nil {
		return nil, err
	}

	var admissionHooks []hooks.AdmissionHook
	if c.ExtraConfig.EnableValidatingWebhook {
		admissionHooks = append(admissionHooks,
			ctrl.NewResticWebhook(),
			ctrl.NewRecoveryWebhook(),
			ctrl.NewRepositoryWebhook(),
		)
	}
	if c.ExtraConfig.EnableMutatingWebhook {
		admissionHooks = append(admissionHooks,
			ctrl.NewDeploymentWebhook(),
			ctrl.NewDaemonSetWebhook(),
			ctrl.NewStatefulSetWebhook(),
			ctrl.NewReplicationControllerWebhook(),
			ctrl.NewReplicaSetWebhook(),
		)
	}

	s := &StashServer{
		GenericAPIServer: genericServer,
		Controller:       ctrl,
	}

	for _, versionMap := range admissionHooksByGroupThenVersion(admissionHooks...) {
		// TODO we're going to need a later k8s.io/apiserver so that we can get discovery to list a different group version for
		// our endpoint which we'll use to back some custom storage which will consume the AdmissionReview type and give back the correct response
		apiGroupInfo := genericapiserver.APIGroupInfo{
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
				admissionResource, _ := admissionHook.Resource()
				admissionVersion := admissionResource.GroupVersion()

				// just overwrite the groupversion with a random one.  We don't really care or know.
				apiGroupInfo.PrioritizedVersions = appendUniqueGroupVersion(apiGroupInfo.PrioritizedVersions, admissionVersion)

				admissionReview := admissionreview.NewREST(admissionHook.Admit)
				v1alpha1storage, ok := apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version]
				if !ok {
					v1alpha1storage = map[string]rest.Storage{}
				}
				v1alpha1storage[admissionResource.Resource] = admissionReview
				apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version] = v1alpha1storage
			}
		}

		if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return nil, err
		}
	}

	for i := range admissionHooks {
		admissionHook := admissionHooks[i]
		postStartName := postStartHookName(admissionHook)
		if len(postStartName) == 0 {
			continue
		}
		s.GenericAPIServer.AddPostStartHookOrDie(postStartName,
			func(context genericapiserver.PostStartHookContext) error {
				return admissionHook.Initialize(c.ExtraConfig.ClientConfig, context.StopCh)
			},
		)
	}

	if c.ExtraConfig.EnableValidatingWebhook {
		s.GenericAPIServer.AddPostStartHookOrDie("validating-webhook-xray",
			func(context genericapiserver.PostStartHookContext) error {
				go func() {
					xray := reg_util.NewCreateValidatingWebhookXray(c.ExtraConfig.ClientConfig, apiserviceName, &api.Repository{
						TypeMeta: metav1.TypeMeta{
							APIVersion: api.SchemeGroupVersion.String(),
							Kind:       api.ResourceKindRepository,
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-repository-for-webhook-xray",
							Namespace: "default",
						},
						Spec: api.RepositorySpec{
							WipeOut: true,
							Backend: store.Backend{
								Local: &store.LocalSpec{
									VolumeSource: core.VolumeSource{
										HostPath: &core.HostPathVolumeSource{
											Path: "/tmp/test",
										},
									},
									MountPath: "/tmp/test",
								},
							},
						},
					}, context.StopCh)
					if err := xray.IsActive(); err != nil {
						w, _, e2 := dynamic_util.DetectWorkload(
							c.ExtraConfig.ClientConfig,
							core.SchemeGroupVersion.WithResource("pods"),
							os.Getenv("MY_POD_NAMESPACE"),
							os.Getenv("MY_POD_NAME"))
						if e2 == nil {
							eventer.CreateEventWithLog(
								kubernetes.NewForConfigOrDie(c.ExtraConfig.ClientConfig),
								"stash-operator",
								w,
								core.EventTypeWarning,
								eventer.EventReasonAdmissionWebhookNotActivated,
								err.Error())
						}
						panic(err)
					}
				}()
				return nil
			},
		)
	}

	{
		apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(repositories.GroupName, Scheme, metav1.ParameterCodec, Codecs)
		v1alpha1storage := map[string]rest.Storage{}
		v1alpha1storage[v1alpha1.ResourcePluralSnapshot] = snapregistry.NewREST(c.ExtraConfig.ClientConfig)
		apiGroupInfo.VersionedResourcesStorageMap["v1alpha1"] = v1alpha1storage

		if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func appendUniqueGroupVersion(slice []schema.GroupVersion, elems ...schema.GroupVersion) []schema.GroupVersion {
	m := map[schema.GroupVersion]bool{}
	for _, gv := range slice {
		m[gv] = true
	}
	for _, e := range elems {
		m[e] = true
	}
	out := make([]schema.GroupVersion, 0, len(m))
	for gv := range m {
		out = append(out, gv)
	}
	return out
}

func postStartHookName(hook hooks.AdmissionHook) string {
	var ns []string
	gvr, _ := hook.Resource()
	ns = append(ns, fmt.Sprintf("admit-%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group))
	if len(ns) == 0 {
		return ""
	}
	return strings.Join(append(ns, "init"), "-")
}

func admissionHooksByGroupThenVersion(admissionHooks ...hooks.AdmissionHook) map[string]map[string][]hooks.AdmissionHook {
	ret := map[string]map[string][]hooks.AdmissionHook{}
	for i := range admissionHooks {
		hook := admissionHooks[i]
		gvr, _ := hook.Resource()
		group, ok := ret[gvr.Group]
		if !ok {
			group = map[string][]hooks.AdmissionHook{}
			ret[gvr.Group] = group
		}
		group[gvr.Version] = append(group[gvr.Version], hook)
	}
	return ret
}

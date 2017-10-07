package controller

import (
	"time"

	sapi "github.com/appscode/stash/apis/stash"
	sapi_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	scs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type Controller struct {
	kubeClient      clientset.Interface
	stashClient     scs.ResticsGetter
	crdClient       apiextensionsclient.Interface
	SidecarImageTag string
	resyncPeriod    time.Duration
}

func New(kubeClient clientset.Interface, crdClient apiextensionsclient.Interface, extClient scs.ResticsGetter, tag string, resyncPeriod time.Duration) *Controller {
	return &Controller{
		kubeClient:      kubeClient,
		stashClient:     extClient,
		crdClient:       crdClient,
		SidecarImageTag: tag,
		resyncPeriod:    resyncPeriod,
	}
}

func (c *Controller) Setup() error {
	if err := c.ensureCustomResourceDefinitions(); err != nil {
		return err
	}

	return nil
}

func (c *Controller) ensureCustomResourceDefinitions() error {
	crds := []*apiextensions.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   sapi.ResourceTypeRestic + "." + sapi_v1alpha1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   sapi.GroupName,
				Version: sapi_v1alpha1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   sapi.ResourceNameRestic,
					Plural:     sapi.ResourceTypeRestic,
					Kind:       sapi.ResourceKindRestic,
					ShortNames: []string{"rst"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := c.crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = c.crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return util.WaitForCRDReady(
		c.kubeClient.CoreV1().RESTClient(),
		crds,
	)
}

func (c *Controller) Run() {
	go c.WatchNamespaces()
	go c.WatchRestics()
	go c.WatchDaemonSets()
	go c.WatchStatefulSets()
	go c.WatchDeployments()
	go c.WatchReplicaSets()
	go c.WatchReplicationControllers()
}

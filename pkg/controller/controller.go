package controller

import (
	"time"

	tapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/util"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	updateRetryInterval = 10 * 1000 * 1000 * time.Nanosecond
	maxAttempts         = 5
)

type Controller struct {
	kubeClient      clientset.Interface
	stashClient     scs.ExtensionInterface
	crdClient       apiextensionsclient.Interface
	SidecarImageTag string
	syncPeriod      time.Duration
}

func New(kubeClient clientset.Interface, crdClient apiextensionsclient.Interface, extClient scs.ExtensionInterface, tag string) *Controller {
	return &Controller{
		kubeClient:      kubeClient,
		stashClient:     extClient,
		crdClient:       crdClient,
		SidecarImageTag: tag,
		syncPeriod:      30 * time.Second,
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
				Name:   tapi.ResourceTypeRestic + "." + tapi.V1alpha1SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   tapi.GroupName,
				Version: tapi.V1alpha1SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   tapi.ResourceNameRestic,
					Plural:     tapi.ResourceTypeRestic,
					Kind:       tapi.ResourceKindRestic,
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
	go c.WatchDeploymentApps()
	go c.WatchDeploymentExtensions()
	go c.WatchReplicaSets()
	go c.WatchReplicationControllers()
}

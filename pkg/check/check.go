package check

import (
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	namespace   string
	resticName  string
	hostName    string
	smartPrefix string
	recorder    record.EventRecorder
}

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, namespace, name, host, prefix string) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		namespace:   namespace,
		resticName:  name,
		hostName:    host,
		smartPrefix: prefix,
		recorder:    eventer.NewEventRecorder(k8sClient, "stash-check"),
	}
}

func (c *Controller) Run() (err error) {
	restic, err := c.stashClient.Restics(c.namespace).Get(c.resticName, metav1.GetOptions{})
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			c.recorder.Eventf(restic.ObjectReference(), core.EventTypeNormal, eventer.EventReasonFailedToCheck, "Check failed for pod %s, reason: %s\n", c.hostName, err.Error())
		} else {
			c.recorder.Eventf(restic.ObjectReference(), core.EventTypeNormal, eventer.EventReasonSuccessfulCheck, "Check successful for pod: %s\n", c.hostName)
		}
	}()

	secret, err := c.k8sClient.CoreV1().Secrets(c.namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return
	}

	cli := cli.New("/tmp", c.hostName)
	if err = cli.SetupEnv(restic, secret, c.smartPrefix); err != nil {
		return
	}

	err = cli.Check()
	return
}

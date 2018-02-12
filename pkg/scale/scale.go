package scale

import (
	"fmt"
	"strconv"
	"time"

	apps_util "github.com/appscode/kutil/apps/v1beta1"
	"github.com/appscode/stash/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/appscode/go/types"
)

type Options struct {
	Namespace      string
	Label          string
	DockerRegistry string // image registry for check job
	ImageTag       string // image tag for check job
}

type Controller struct {
	k8sClient kubernetes.Interface
	opt       Options
	locked    chan struct{}
}

var (
	ZeroReplica = int32(0)
	OneReplica  = int32(1)
)

func New(k8sClient kubernetes.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient: k8sClient,
		opt:       opt,
	}
}

func (c *Controller) ScaleDownWorkload() error {
	fmt.Println("ScaleDownWorkload called. LabelSelector:",c.opt.Label)
	dpList, err := c.k8sClient.AppsV1beta1().Deployments(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Label})
	fmt.Println("dpList Length:",len(dpList.Items))
	if err == nil {
		for _, dp := range dpList.Items {
			_, _, err := apps_util.PatchDeployment(c.k8sClient, &dp, func(obj *apps.Deployment) *apps.Deployment {
				if obj.Annotations == nil {
					obj.Annotations = make(map[string]string)
				}
				obj.Annotations[util.AnnotationOldReplica] = strconv.Itoa(int(*dp.Spec.Replicas))
				obj.Spec.Replicas = &ZeroReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}
	time.Sleep(time.Second * 30)

	if len(dpList.Items) > 0{
		for _, dp := range dpList.Items {
			_, _, err := apps_util.PatchDeployment(c.k8sClient, &dp, func(obj *apps.Deployment) *apps.Deployment {
				obj.Spec.Replicas = &OneReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}


	return nil
}

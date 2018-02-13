package scale

import (
	"fmt"
	"strconv"
	"time"

	"github.com/appscode/go/types"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/backup"
	"github.com/appscode/stash/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Workload       api.LocalTypedReference
	Namespace      string
	Selector       string
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
	dpList, err := c.k8sClient.AppsV1beta1().Deployments(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	fmt.Println("dpList Length:", len(dpList.Items))
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

	if len(dpList.Items) > 0 {
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

func ScaleUpWorkload(k8sClient *kubernetes.Clientset, opt backup.Options) error {
	switch opt.Workload.Kind {
	case api.KindDeployment:
		obj, err := k8sClient.AppsV1beta1().Deployments(opt.Namespace).Get(opt.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		replica, err := util.GetOriginalReplicaFromAnnotation(opt.Workload.Kind, obj)
		if err != nil {
			return err
		}

		_, _, err = apps_util.PatchDeployment(k8sClient, obj, func(dp *apps.Deployment) *apps.Deployment {
			dp.Spec.Replicas = types.Int32P(replica)
			delete(dp.Annotations, util.AnnotationOldReplica)
			return dp
		})
		if err != nil {
			return err
		}
	case api.KindReplicationController:
		//todo
	case api.KindReplicaSet:
		//todo

	}

	return nil
}

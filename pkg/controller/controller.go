package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/log"
	rapi "github.com/appscode/restik/api"
	tcs "github.com/appscode/restik/client/clientset"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func NewRestikController(c *rest.Config, image string) *Controller {
	return &Controller{
		ExtClientset: tcs.NewForConfigOrDie(c),
		Clientset:    clientset.NewForConfigOrDie(c),
		SyncPeriod:   time.Minute * 2,
		Image:        image,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) RunAndHold() error {
	if err := c.ensureResource(); err != nil {
		return err
	}
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.ExtClientset.Restiks(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.ExtClientset.Restiks(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&rapi.Restik{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Restik); ok {
					glog.Infoln("Got one added Restik obejct", b)
					if b.ObjectMeta.Annotations != nil {
						_, ok := b.ObjectMeta.Annotations[ImageAnnotation]
						if ok {
							glog.Infoln("Got one added Restik obejct that was previously deployed", b)
							return
						}
					}
					err := c.updateObjectAndStartBackup(b)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Restik); ok {
					glog.Infoln("Got one deleted Restik object", b)
					err := c.updateObjectAndStopBackup(b)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*rapi.Restik)
				if !ok {
					log.Errorln(errors.New("Error validating Restik object"))
					return
				}
				newObj, ok := new.(*rapi.Restik)
				if !ok {
					log.Errorln(errors.New("Error validating Restik object"))
					return
				}
				var oldImage, newImage string
				if oldObj.ObjectMeta.Annotations != nil {
					oldImage, _ = oldObj.ObjectMeta.Annotations[ImageAnnotation]
				}
				if newObj.ObjectMeta.Annotations != nil {
					newImage, _ = newObj.ObjectMeta.Annotations[ImageAnnotation]
				}
				if oldImage != newImage {
					glog.Infoln("Got one updated Restik object for image", newObj)
					err := c.updateImage(newObj, newImage)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
	return nil
}

func (c *Controller) updateObjectAndStartBackup(r *rapi.Restik) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	restikContainer := getRestikContainer(r, c.Image)
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := metav1.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &apiv1.ReplicationController{}
		if err = yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, restikContainer)
		rc.Spec.Template.Spec.Volumes = append(rc.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		newRC, err := c.Clientset.CoreV1().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels).String()
		if err = restartPods(c.Clientset, r.Namespace, opts); err != nil {
			return err
		}
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err = yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, restikContainer)
		replicaset.Spec.Template.Spec.Volumes = append(replicaset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		newReplicaset, err := c.Clientset.ExtensionsV1beta1().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels).String()
		if err = restartPods(c.Clientset, r.Namespace, opts); err != nil {
			return err
		}
	case Deployment:
		deployment := &extensions.Deployment{}
		if err = yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, restikContainer)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		_, err = c.Clientset.ExtensionsV1beta1().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = append(daemonset.Spec.Template.Spec.Containers, restikContainer)
		daemonset.Spec.Template.Spec.Volumes = append(daemonset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		newDaemonset, err := c.Clientset.ExtensionsV1beta1().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels).String()
		if err = restartPods(c.Clientset, r.Namespace, opts); err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred by the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	c.addAnnotation(r)
	_, err = c.ExtClientset.Restiks(r.Namespace).Update(r)
	return err
}

func (c *Controller) updateObjectAndStopBackup(r *rapi.Restik) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := metav1.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &apiv1.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = removeContainer(rc.Spec.Template.Spec.Containers, ContainerName)
		rc.Spec.Template.Spec.Volumes = removeVolume(rc.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newRC, err := c.Clientset.Core().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = removeContainer(replicaset.Spec.Template.Spec.Containers, ContainerName)
		replicaset.Spec.Template.Spec.Volumes = removeVolume(replicaset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newReplicaset, err := c.Clientset.ExtensionsV1beta1().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = removeContainer(daemonset.Spec.Template.Spec.Containers, ContainerName)
		daemonset.Spec.Template.Spec.Volumes = removeVolume(daemonset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newDaemonset, err := c.Clientset.ExtensionsV1beta1().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = removeContainer(deployment.Spec.Template.Spec.Containers, ContainerName)
		deployment.Spec.Template.Spec.Volumes = removeVolume(deployment.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		_, err := c.Clientset.ExtensionsV1beta1().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred by the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

func (c *Controller) updateImage(r *rapi.Restik, image string) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := metav1.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &apiv1.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = updateImageForRestikContainer(rc.Spec.Template.Spec.Containers, ContainerName, image)
		newRC, err := c.Clientset.Core().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = updateImageForRestikContainer(replicaset.Spec.Template.Spec.Containers, ContainerName, image)
		newReplicaset, err := c.Clientset.ExtensionsV1beta1().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = updateImageForRestikContainer(daemonset.Spec.Template.Spec.Containers, ContainerName, image)
		newDaemonset, err := c.Clientset.ExtensionsV1beta1().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels).String()
		return restartPods(c.Clientset, r.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = updateImageForRestikContainer(deployment.Spec.Template.Spec.Containers, ContainerName, image)
		_, err := c.Clientset.Extensions().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred bt the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

func (c *Controller) ensureResource() error {
	_, err := c.Clientset.ExtensionsV1beta1().ThirdPartyResources().Get(tcs.ResourceNameRestik+"."+rapi.GroupName, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		tpr := &extensions.ThirdPartyResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ThirdPartyResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tcs.ResourceNameRestik + "." + rapi.GroupName,
			},
			Versions: []extensions.APIVersion{
				{
					Name: "v1alpha1",
				},
			},
		}
		_, err := c.Clientset.Extensions().ThirdPartyResources().Create(tpr)
		if err != nil {
			// This should fail if there is one third party resource data missing.
			return err
		}
	}
	return nil
}

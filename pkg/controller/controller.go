package controller

import (
	"errors"
	"fmt"
	"time"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	rapi "github.com/appscode/stash/api"
	rcs "github.com/appscode/stash/client/clientset"
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
	"k8s.io/client-go/tools/cache"
)

const (
	ContainerName      = "stash"
	StashNamespace    = "STASH_NAMESPACE"
	StashResourceName = "STASH_RESOURCE_NAME"

	BackupConfig          = "stash.appscode.com/config"
	RESTIC_PASSWORD       = "RESTIC_PASSWORD"
	ReplicationController = "ReplicationController"
	ReplicaSet            = "ReplicaSet"
	Deployment            = "Deployment"
	DaemonSet             = "DaemonSet"
	StatefulSet           = "StatefulSet"
	ImageAnnotation       = "stash.appscode.com/image"
	Force                 = "force"
)

type Controller struct {
	ExtClientset rcs.ExtensionInterface
	Clientset    clientset.Interface
	// sync time to sync the list.
	SyncPeriod time.Duration
	// image of sidecar container
	SidecarImageTag string
}

func NewController(kubeClient clientset.Interface, extClient rcs.ExtensionInterface, tag string) *Controller {
	return &Controller{
		Clientset:       kubeClient,
		ExtClientset:    extClient,
		SidecarImageTag: tag,
		SyncPeriod:      time.Minute * 2,
	}
}

func (c *Controller) Setup() error {
	_, err := c.Clientset.ExtensionsV1beta1().ThirdPartyResources().Get(rcs.ResourceNameStash+"."+rapi.GroupName, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		tpr := &extensions.ThirdPartyResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ThirdPartyResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: rcs.ResourceNameStash + "." + rapi.GroupName,
			},
			Versions: []extensions.APIVersion{
				{
					Name: "v1alpha1",
				},
			},
		}
		_, err := c.Clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
		if err != nil {
			// This should fail if there is one third party resource data missing.
			return err
		}
	}
	return nil
}

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) RunAndHold() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.ExtClientset.Stashs(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.ExtClientset.Stashs(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&rapi.Restic{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Restic); ok {
					glog.Infoln("Got one added Stash obejct", b)
					if b.ObjectMeta.Annotations != nil {
						_, ok := b.ObjectMeta.Annotations[ImageAnnotation]
						if ok {
							glog.Infoln("Got one added Stash obejct that was previously deployed", b)
							return
						}
					}
					err := c.updateObjectAndStartBackup(b)
					if err != nil {
						sidecarFailedToAdd()
						log.Errorln(err)
					} else {
						sidecarSuccessfullyAdd()
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Restic); ok {
					glog.Infoln("Got one deleted Stash object", b)
					err := c.updateObjectAndStopBackup(b)
					if err != nil {
						sidecarFailedToDelete()
						log.Errorln(err)
					} else {
						sidecarSuccessfullyDeleted()
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*rapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				newObj, ok := new.(*rapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				var oldImage, newImage string
				if oldObj.ObjectMeta.Annotations != nil {
					oldImage = oldObj.ObjectMeta.Annotations[ImageAnnotation]
				}
				if newObj.ObjectMeta.Annotations != nil {
					newImage = newObj.ObjectMeta.Annotations[ImageAnnotation]
				}
				if oldImage != newImage {
					glog.Infoln("Got one updated Stash object for image", newObj)
					err := c.updateImage(newObj, newImage)
					if err != nil {
						sidecarFailedToUpdate()
						log.Errorln(err)
					} else {
						sidecarSuccessfullyUpdated()
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) updateObjectAndStartBackup(r *rapi.Restic) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Stash %s ", r.Name))
	}
	opts := metav1.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &apiv1.ReplicationController{}
		if err = yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, c.GetSidecarContainer(r))
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
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, c.GetSidecarContainer(r))
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
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, c.GetSidecarContainer(r))
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
		daemonset.Spec.Template.Spec.Containers = append(daemonset.Spec.Template.Spec.Containers, c.GetSidecarContainer(r))
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
		log.Warningf("The Object referred by the Stash object (%s) is a statefulset.", r.Name)
		return nil
	}
	c.addAnnotation(r)
	_, err = c.ExtClientset.Stashs(r.Namespace).Update(r)
	return err
}

func (c *Controller) updateObjectAndStopBackup(r *rapi.Restic) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Stash %s ", r.Name))
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
		newRC, err := c.Clientset.CoreV1().ReplicationControllers(r.Namespace).Update(rc)
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
		log.Warningf("The Object referred by the Stash object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

func (c *Controller) updateImage(r *rapi.Restic, image string) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(c.Clientset, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Stash %s ", r.Name))
	}
	opts := metav1.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &apiv1.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = updateImageForStashContainer(rc.Spec.Template.Spec.Containers, ContainerName, image)
		newRC, err := c.Clientset.CoreV1().ReplicationControllers(r.Namespace).Update(rc)
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
		replicaset.Spec.Template.Spec.Containers = updateImageForStashContainer(replicaset.Spec.Template.Spec.Containers, ContainerName, image)
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
		daemonset.Spec.Template.Spec.Containers = updateImageForStashContainer(daemonset.Spec.Template.Spec.Containers, ContainerName, image)
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
		deployment.Spec.Template.Spec.Containers = updateImageForStashContainer(deployment.Spec.Template.Spec.Containers, ContainerName, image)
		_, err := c.Clientset.ExtensionsV1beta1().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred bt the Stash object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

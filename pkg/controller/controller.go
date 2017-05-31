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
	"k8s.io/kubernetes/pkg/api"
	k8serrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

func NewRestikController(c *rest.Config, image string) *Controller {
	return &Controller{
		ExtClient:  tcs.NewACExtensionsForConfigOrDie(c),
		Client:     clientset.NewForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
		Image:      image,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Controller) RunAndHold() error {
	if err := w.ensureResource(); err != nil {
		return err
	}
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return w.ExtClient.Restiks(api.NamespaceAll).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return w.ExtClient.Restiks(api.NamespaceAll).Watch(api.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&rapi.Restik{},
		w.SyncPeriod,
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
					err := w.updateObjectAndStartBackup(b)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Restik); ok {
					glog.Infoln("Got one deleted Restik object", b)
					err := w.updateObjectAndStopBackup(b)
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
					err := w.updateImage(newObj, newImage)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
	return nil
}

func (pl *Controller) updateObjectAndStartBackup(r *rapi.Restik) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	restikContainer := getRestikContainer(r, pl.Image)
	ob, typ, err := getKubeObject(pl.Client, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err = yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, restikContainer)
		rc.Spec.Template.Spec.Volumes = append(rc.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		newRC, err := pl.Client.Core().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		if err = restartPods(pl.Client, r.Namespace, opts); err != nil {
			return err
		}
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err = yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, restikContainer)
		replicaset.Spec.Template.Spec.Volumes = append(replicaset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		if err = restartPods(pl.Client, r.Namespace, opts); err != nil {
			return err
		}
	case Deployment:
		deployment := &extensions.Deployment{}
		if err = yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, restikContainer)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume)
		_, err = pl.Client.Extensions().Deployments(r.Namespace).Update(deployment)
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
		newDaemonset, err := pl.Client.Extensions().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		if err = restartPods(pl.Client, r.Namespace, opts); err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred by the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	pl.addAnnotation(r)
	_, err = pl.ExtClient.Restiks(r.Namespace).Update(r)
	return err
}

func (pl *Controller) updateObjectAndStopBackup(r *rapi.Restik) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(pl.Client, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = removeContainer(rc.Spec.Template.Spec.Containers, ContainerName)
		rc.Spec.Template.Spec.Volumes = removeVolume(rc.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newRC, err := pl.Client.Core().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = removeContainer(replicaset.Spec.Template.Spec.Containers, ContainerName)
		replicaset.Spec.Template.Spec.Volumes = removeVolume(replicaset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = removeContainer(daemonset.Spec.Template.Spec.Containers, ContainerName)
		daemonset.Spec.Template.Spec.Volumes = removeVolume(daemonset.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		newDaemonset, err := pl.Client.Extensions().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = removeContainer(deployment.Spec.Template.Spec.Containers, ContainerName)
		deployment.Spec.Template.Spec.Volumes = removeVolume(deployment.Spec.Template.Spec.Volumes, r.Spec.Destination.Volume.Name)
		_, err := pl.Client.Extensions().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred by the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

func (pl *Controller) updateImage(r *rapi.Restik, image string) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: r.Name})
	ob, typ, err := getKubeObject(pl.Client, r.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for Restik %s ", r.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = updateImageForRestikContainer(rc.Spec.Template.Spec.Containers, ContainerName, image)
		newRC, err := pl.Client.Core().ReplicationControllers(r.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = updateImageForRestikContainer(replicaset.Spec.Template.Spec.Containers, ContainerName, image)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(r.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = updateImageForRestikContainer(daemonset.Spec.Template.Spec.Containers, ContainerName, image)
		newDaemonset, err := pl.Client.Extensions().DaemonSets(r.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		return restartPods(pl.Client, r.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = updateImageForRestikContainer(deployment.Spec.Template.Spec.Containers, ContainerName, image)
		_, err := pl.Client.Extensions().Deployments(r.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred bt the Restik object (%s) is a statefulset.", r.Name)
		return nil
	}
	return nil
}

func (w *Controller) ensureResource() error {
	_, err := w.Client.Extensions().ThirdPartyResources().Get(tcs.ResourceNameRestik + "." + rapi.GroupName)
	if k8serrors.IsNotFound(err) {
		tpr := &extensions.ThirdPartyResource{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ThirdPartyResource",
			},
			ObjectMeta: api.ObjectMeta{
				Name: tcs.ResourceNameRestik + "." + rapi.GroupName,
			},
			Versions: []extensions.APIVersion{
				{
					Name: "v1beta1",
				},
			},
		}
		_, err := w.Client.Extensions().ThirdPartyResources().Create(tpr)
		if err != nil {
			// This should fail if there is one third party resource data missing.
			return err
		}
	}
	return nil
}

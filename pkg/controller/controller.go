package controller

import (
	"errors"
	"fmt"
	"time"

	rapi "github.com/appscode/k8s-addons/api"
	tcs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/log"
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
	"path/filepath"
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
			return w.ExtClient.Backups(api.NamespaceAll).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return w.ExtClient.Backups(api.NamespaceAll).Watch(api.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&rapi.Backup{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Backup); ok {
					glog.Infoln("Got one added bacup obejct", b)
					if b.ObjectMeta.Annotations != nil {
						_, ok := b.ObjectMeta.Annotations[ImageAnnotation]
						if ok {
							glog.Infoln("Got one added backup obejct that was previously deployed", b)
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
				if b, ok := obj.(*rapi.Backup); ok {
					glog.Infoln("Got one deleted backup object", b)
					err := w.updateObjectAndStopBackup(b)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*rapi.Backup)
				if !ok {
					log.Errorln(errors.New("Error validating backup object"))
					return
				}
				newObj, ok := new.(*rapi.Backup)
				if !ok {
					log.Errorln(errors.New("Error validating backup object"))
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
					glog.Infoln("Got one updated backp object for image", newObj)
					err := w.updateImage(newObj, newImage)
					if err != nil {
						log.Errorln(err)
					}
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
}

func RunBackup() {
	factory := cmdutil.NewFactory(nil)
	config, err := factory.ClientConfig()
	if err != nil {
		log.Errorln(err)
		return
	}
	client, err := factory.ClientSet()
	if err != nil {
		log.Errorln(err)
		return
	}
	cronWatcher := &cronController{
		extClient:     tcs.NewACExtensionsForConfigOrDie(config),
		kubeClient:    client,
		namespace:     os.Getenv(RestikNamespace),
		tprName:       os.Getenv(RestikResourceName),
		crons:         cron.New(),
		eventRecorder: NewEventRecorder(client, "Restik sidecar Watcher"),
	}
	cronWatcher.crons.Start()
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return cronWatcher.extClient.Backups(cronWatcher.namespace).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return cronWatcher.extClient.Backups(cronWatcher.namespace).Watch(api.ListOptions{})
		},
	}
	_, cronController := cache.NewInformer(lw,
		&rapi.Backup{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Backup); ok {
					if b.Name == cronWatcher.tprName {
						cronWatcher.backup = b
						err = cronWatcher.startCron()
						if err != nil {
							log.Errorln(err)
							return
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*rapi.Backup)
				if !ok {
					return
				}
				newObj, ok := new.(*rapi.Backup)
				if !ok {
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == cronWatcher.tprName {
					cronWatcher.backup = newObj
					err = cronWatcher.startCron()
					if err != nil {
						log.Errorln(err)
						return
					}
				}
			},
		})
	cronController.Run(wait.NeverStop)
}

func (cronWatcher *cronController) startCron() error {
	backup := cronWatcher.backup
	password, err := getPasswordFromSecret(cronWatcher.kubeClient, backup.Spec.Destination.RepositorySecretName, backup.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
	}
	repo := backup.Spec.Destination.Path
	_, err = os.Stat(filepath.Join(repo, "config"))
	if os.IsNotExist(err) {
		if _, err = execLocal(fmt.Sprintf("/restic init --repo %s", repo)); err != nil {
			return err
		}
	}
	// Remove previous jobs
	for _, v := range cronWatcher.crons.Entries() {
		cronWatcher.crons.Remove(v.ID)
	}
	interval := backup.Spec.Schedule
	if _, err = cron.Parse(interval); err != nil {
		er := err
		//Reset Wrong Schedule
		backup.Spec.Schedule = ""
		// Create event
		cronWatcher.eventRecorder.Event(backup, api.EventTypeWarning, EventReasonCronExpressionFailed, err.Error())
		_, err = cronWatcher.extClient.Backups(backup.Namespace).Update(backup)
		if err != nil {
			log.Errorln(err)
		}
		return er
	}
	_, err = cronWatcher.crons.AddFunc(interval, cronWatcher.runCronJob)
	if err != nil {
		return err
	}
	return nil
}

func (pl *Controller) updateObjectAndStartBackup(b *rapi.Backup) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: b.Name})
	restikContainer := getRestikContainer(b, pl.Image)
	ob, typ, err := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, restikContainer)
		rc.Spec.Template.Spec.Volumes = append(rc.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newRC, err := pl.Client.Core().ReplicationControllers(b.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		err = restartPods(pl.Client, b.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, restikContainer)
		replicaset.Spec.Template.Spec.Volumes = append(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		err = restartPods(pl.Client, b.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, restikContainer)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		_, err := pl.Client.Extensions().Deployments(b.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = append(daemonset.Spec.Template.Spec.Containers, restikContainer)
		daemonset.Spec.Template.Spec.Volumes = append(daemonset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newDaemonset, err := pl.Client.Extensions().DaemonSets(b.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		err = restartPods(pl.Client, b.Namespace, opts)
	case StatefulSet:
		log.Warningf("The Object referred by the backup object (%s) is a statefulset.", b.Name)
		return nil
	}
	if err != nil {
		return err
	}
	pl.addAnnotation(b)
	_, err = pl.ExtClient.Backups(b.Namespace).Update(b)
	return err
}

func (pl *Controller) updateObjectAndStopBackup(b *rapi.Backup) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: b.Name})
	ob, typ, err := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = removeContainer(rc.Spec.Template.Spec.Containers, ContainerName)
		rc.Spec.Template.Spec.Volumes = removeVolume(rc.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newRC, err := pl.Client.Core().ReplicationControllers(b.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = removeContainer(replicaset.Spec.Template.Spec.Containers, ContainerName)
		replicaset.Spec.Template.Spec.Volumes = removeVolume(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = removeContainer(daemonset.Spec.Template.Spec.Containers, ContainerName)
		daemonset.Spec.Template.Spec.Volumes = removeVolume(daemonset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newDaemonset, err := pl.Client.Extensions().DaemonSets(b.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = removeContainer(deployment.Spec.Template.Spec.Containers, ContainerName)
		deployment.Spec.Template.Spec.Volumes = removeVolume(deployment.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		_, err := pl.Client.Extensions().Deployments(b.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred bt the backup object (%s) is a statefulset.", b.Name)
		return nil
	}
	return nil
}

func (pl *Controller) updateImage(b *rapi.Backup, image string) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: b.Name})
	ob, typ, err := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || typ == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
	}
	opts := api.ListOptions{}
	switch typ {
	case ReplicationController:
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = updateImageForRestikContainer(rc.Spec.Template.Spec.Containers, ContainerName, image)
		newRC, err := pl.Client.Core().ReplicationControllers(b.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = updateImageForRestikContainer(replicaset.Spec.Template.Spec.Containers, ContainerName, image)
		newReplicaset, err := pl.Client.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = updateImageForRestikContainer(daemonset.Spec.Template.Spec.Containers, ContainerName, image)
		newDaemonset, err := pl.Client.Extensions().DaemonSets(b.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		return restartPods(pl.Client, b.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = updateImageForRestikContainer(deployment.Spec.Template.Spec.Containers, ContainerName, image)
		_, err := pl.Client.Extensions().Deployments(b.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		log.Warningf("The Object referred bt the backup object (%s) is a statefulset.", b.Name)
		return nil
	}
	return nil
}

func (w *Controller) ensureResource() error {
	_, err := w.Client.Extensions().ThirdPartyResources().Get(tcs.ResourceNameBackup + "." + rapi.GroupName)
	if k8serrors.IsNotFound(err) {
		tpr := &extensions.ThirdPartyResource{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ThirdPartyResource",
			},
			ObjectMeta: api.ObjectMeta{
				Name: tcs.ResourceNameBackup + "." + rapi.GroupName,
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

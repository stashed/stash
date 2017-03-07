package controller

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	rapi "github.com/appscode/restik/api"
	tcs "github.com/appscode/restik/client/clientset"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	//"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	Client tcs.ExtensionInterface
	// sync time to sync the list.
	SyncPeriod time.Duration
}

func New(c *rest.Config) *Controller {
	return &Controller{
		Client:     tcs.NewExtensionsForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Controller) RunAndHold() {
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return w.Client.Backup(api.NamespaceAll).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return w.Client.Backup(api.NamespaceAll).Watch(api.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&rapi.Backup{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.Infoln("Got one added bacup obejct", obj.(*rapi.Backup))
				err := w.updateObjectAndBackup(obj.(*rapi.Backup))
				if err != nil {
					fmt.Println(err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				glog.Infoln("Got one deleted backu object", obj.(*rapi.Backup))
				w.doStuff(obj.(*rapi.Backup))
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
					glog.Infoln("Got one updated backp object", newObj)
					w.doStuff(newObj)
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
}

func (pl *Controller) doStuff(release *rapi.Backup) {

}

func (pl *Controller) updateObjectAndBackup(b *rapi.Backup) error {
	factory := cmdutil.NewFactory(nil)
	kubeClient, err := factory.ClientSet()
	if err != nil {
		return err
	}
	set := labels.Set{
		BackupConfig: b.Name,
	}
	///*fieldSelector*/ fields.SelectorFromSet(set)
	ls := labels.SelectorFromSet(set)
	opts := api.ListOptions{
		//FieldSelector: fieldSelector,
		LabelSelector: ls,
	}
	//fmt.Println(fieldSelector)
	fmt.Println(b.Namespace)
	//TODO make work on different objects
	rcs, err := kubeClient.Core().ReplicationControllers(b.Namespace).List(opts)
	if err != nil {
		return err
	}
	if len(rcs.Items) == 0 {
		return errors.New("No RC found")
	}
	rc := rcs.Items[0]
	restikContainer := getRestikContainer(b)
	rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, restikContainer)
	rc.Spec.Template.Spec.Volumes = append(rc.Spec.Template.Spec.Volumes, b.Spec.Destination)
	newRC, err := kubeClient.Core().ReplicationControllers(b.Namespace).Update(&rc)
	if err != nil {
		return err
	}
	selectors := getSelectors(newRC)
	if err != nil {
		return err
	}
	opts = api.ListOptions{
		LabelSelector: selectors,
	}
	pods, err := kubeClient.Core().Pods(b.Namespace).List(opts)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		deleteOpts := &api.DeleteOptions{}
		err = kubeClient.Core().Pods(b.Namespace).Delete(pod.Name, deleteOpts)
		if err != nil {
			errMessage := fmt.Sprint("Failed to restart pod %s cause %v", pod.Name, err)
			log.Println(errMessage)
		}
	}

	return nil
}

func getRestikContainer(b *rapi.Backup) api.Container {
	container := api.Container{
		Name:            ContainerName,
		Image:           Image,
		ImagePullPolicy: api.PullAlways,
	}
	env := api.EnvVar{
		Name:  "BACKUP_CRON",
		Value: b.Spec.Schedule,
	}
	container.Env = append(container.Env, env)
	env = api.EnvVar{
		Name:  "RESTIC_REPOSITORY",
		Value: RestickMountPath,
	}
	container.Env = append(container.Env, env)
	env = api.EnvVar{
		Name:  "SOURCE_PATH",
		Value: b.Spec.Source.Path,
	}
	container.Env = append(container.Env, env)

	env = api.EnvVar{
		Name:  "RESTIC_PASSWORD",
		Value: "123",
	}
	container.Env = append(container.Env, env)

	backupVolumeMount := api.VolumeMount{
		Name:      VolumeName,
		MountPath: RestickMountPath,
	}
	sourceVolumeMount := api.VolumeMount{
		Name:      b.Spec.Source.VolumeName,
		MountPath: b.Spec.Source.Path,
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	return container
}

func getSelectors(newRC *api.ReplicationController) labels.Selector {
	lb := newRC.Spec.Template.Labels
	set := labels.Set(lb)
	selectores := labels.SelectorFromSet(set)
	return selectores
}

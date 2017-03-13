package controller

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	rapi "github.com/appscode/restik/api"
	tcs "github.com/appscode/restik/client/clientset"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/restic/restic/src/restic/errors"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
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
	// image of sidecar container
	Image string
}

func New(c *rest.Config, image string) *Controller {
	return &Controller{
		Client:     tcs.NewExtensionsForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
		Image:      image,
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
				err := w.updateObjectAndStartBackup(obj.(*rapi.Backup))
				if err != nil {
					log.Println(err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				glog.Infoln("Got one deleted backu object", obj.(*rapi.Backup))
				err := w.updateObjectAndStopBackup(obj.(*rapi.Backup))
				if err != nil {
					log.Println(err)
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
					glog.Infoln("Got one updated backp object", newObj)
					w.doStuff(newObj)
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
		log.Println(err)
		return
	}
	extClient := tcs.NewExtensionsForConfigOrDie(config)
	client, err := factory.ClientSet()
	if err != nil {
		log.Println(err)
		return
	}
	namespace := os.Getenv(Namespace)
	tprName := os.Getenv(TPR)
	backup, err := extClient.Backup(namespace).Get(tprName)
	if err != nil {
		log.Println(err)
		return
	}
	password, err := getPasswordFromSecret(client, backup.Spec.BackupSecretRef, backup.Namespace)
	if err != nil {
		log.Println(err)
		return
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		log.Println(err)
		return
	}
	repo := backup.Spec.Destination.Path
	_, err = os.Stat(filepath.Join(repo, "config"))
	if os.IsNotExist(err) {
		_, err := execLocal(fmt.Sprintf("/restic init --repo %s", repo))
		if err != nil {
			log.Println("RESTIC repository not created cause", err)
			return
		}
	}
	interval := backup.Spec.Schedule
	c := cron.New()
	c.Start()
	c.AddFunc(interval, func() {
		backup, err := extClient.Backup(namespace).Get(tprName)
		if err != nil {
			log.Println(err)
		}
		password, err := getPasswordFromSecret(client, backup.Spec.BackupSecretRef, backup.Namespace)
		err = os.Setenv(RESTIC_PASSWORD, password)
		if err != nil {
			log.Println(err)
		}
		cmd := fmt.Sprintf("/restic -r %s backup %s", backup.Spec.Destination.Path, backup.Spec.Source.Path)
		// add tags if any
		for _, t := range backup.Spec.Tags {
			cmd = cmd + " --tag " + t
		}
		// Take Backup
		_, err = execLocal(cmd)
		if err != nil {
			log.Println("Restick backup failed cause ", err)
		}
		err = snapshotRetention(backup)
		if err != nil {
			log.Println("Snapshot retention failed cause ", err)
		}
	})
	wait.Until(func() {}, time.Second, wait.NeverStop)
}

func (pl *Controller) doStuff(release *rapi.Backup) {

}

func getRestikContainer(b *rapi.Backup, containerImage string) api.Container {
	container := api.Container{
		Name:            ContainerName,
		Image:           containerImage,
		ImagePullPolicy: api.PullAlways,
	}
	env := []api.EnvVar{
		{
			Name:  Namespace,
			Value: b.Namespace,
		},
		{
			Name:  TPR,
			Value: b.Name,
		},
	}
	for _, e := range env {
		container.Env = append(container.Env, e)
	}
	container.Args = append(container.Args, "watch")
	container.Args = append(container.Args, "--v=10")
	backupVolumeMount := api.VolumeMount{
		Name:      b.Spec.Destination.Volume.Name,
		MountPath: b.Spec.Destination.Path,
	}
	sourceVolumeMount := api.VolumeMount{
		Name:      b.Spec.Source.VolumeName,
		MountPath: b.Spec.Source.Path,
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	return container
}

func restartPods(kubeClient *internalclientset.Clientset, namespace string, opts api.ListOptions) error {
	pods, err := kubeClient.Core().Pods(namespace).List(opts)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		deleteOpts := &api.DeleteOptions{}
		err = kubeClient.Core().Pods(namespace).Delete(pod.Name, deleteOpts)
		if err != nil {
			errMessage := fmt.Sprint("Failed to restart pod %s cause %v", pod.Name, err)
			log.Println(errMessage)
		}
	}
	return nil
}

func getKubeObject(kubeClient *internalclientset.Clientset, destination api.Volume, namespace string, ls labels.Selector, restikContainer api.Container) map[string]interface{} {
	uslist := &runtime.UnstructuredList{}
	err := kubeClient.Core().RESTClient().Get().Resource("replicationcontrollers").Namespace(namespace).LabelsSelectorParam(ls).Do().Into(uslist)
	if err == nil && len(uslist.Items) != 0 {
		return uslist.Items[0].Object
	}

	err = kubeClient.Extensions().RESTClient().Get().Resource("replicasets").Namespace(namespace).LabelsSelectorParam(ls).Do().Into(uslist)
	if err == nil && len(uslist.Items) != 0 {
		return uslist.Items[0].Object
	}

	err = kubeClient.Extensions().RESTClient().Get().Resource("deployments").Namespace(namespace).LabelsSelectorParam(ls).Do().Into(uslist)
	if err == nil && len(uslist.Items) != 0 {
		return uslist.Items[0].Object
	}

	err = kubeClient.Extensions().RESTClient().Get().Resource("daemonsets").Namespace(namespace).LabelsSelectorParam(ls).Do().Into(uslist)
	if err == nil && len(uslist.Items) != 0 {
		return uslist.Items[0].Object
	}

	err = kubeClient.Apps().RESTClient().Get().Resource("statefulsets").Namespace(namespace).LabelsSelectorParam(ls).Do().Into(uslist)
	if err == nil && len(uslist.Items) != 0 {
		return uslist.Items[0].Object
	}
	return nil
}

func findSelectors(lb map[string]string) labels.Selector {
	set := labels.Set(lb)
	selectores := labels.SelectorFromSet(set)
	return selectores
}

func (pl *Controller) updateObjectAndStartBackup(b *rapi.Backup) error {
	factory := cmdutil.NewFactory(nil)
	kubeClient, err := factory.ClientSet()
	if err != nil {
		return err
	}
	set := labels.Set{
		BackupConfig: b.Name,
	}
	ls := labels.SelectorFromSet(set)
	restikContainer := getRestikContainer(b, pl.Image)
	object := getKubeObject(kubeClient, b.Spec.Destination.Volume, b.Namespace, ls, restikContainer)
	ob, err := yaml.Marshal(object)
	if err != nil {
		return err
	}
	_type, ok := object["kind"].(string)
	if !ok {
		return nil
	}
	opts := api.ListOptions{}
	switch _type {
	case "ReplicationController":
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = append(rc.Spec.Template.Spec.Containers, restikContainer)
		rc.Spec.Template.Spec.Volumes = append(rc.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newRC, err := kubeClient.Core().ReplicationControllers(b.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)

	case "ReplicaSet":
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, restikContainer)
		replicaset.Spec.Template.Spec.Volumes = append(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newRC, err := kubeClient.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
	}
	//TODO add other objects
	err = restartPods(kubeClient, b.Namespace, opts)
	return err
}

func (pl *Controller) updateObjectAndStopBackup(b *rapi.Backup) error {
	factory := cmdutil.NewFactory(nil)
	kubeClient, err := factory.ClientSet()
	if err != nil {
		return err
	}
	set := labels.Set{
		BackupConfig: b.Name,
	}
	ls := labels.SelectorFromSet(set)
	restikContainer := getRestikContainer(b, pl.Image)
	object := getKubeObject(kubeClient, b.Spec.Destination.Volume, b.Namespace, ls, restikContainer)
	ob, err := yaml.Marshal(object)
	if err != nil {
		return err
	}
	_type, ok := object["kind"].(string)
	if !ok {
		return nil
	}
	opts := api.ListOptions{}
	switch _type {
	case "ReplicationController":
		rc := &api.ReplicationController{}
		if err := yaml.Unmarshal(ob, rc); err != nil {
			return err
		}
		rc.Spec.Template.Spec.Containers = removeContainer(rc.Spec.Template.Spec.Containers, ContainerName)
		rc.Spec.Template.Spec.Volumes = removeVolume(rc.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newRC, err := kubeClient.Core().ReplicationControllers(b.Namespace).Update(rc)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)

	case "ReplicaSet":
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = removeContainer(replicaset.Spec.Template.Spec.Containers, ContainerName)
		replicaset.Spec.Template.Spec.Volumes = removeVolume(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newRC, err := kubeClient.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newRC.Spec.Template.Labels)
	}
	//TODO add other objects
	err = restartPods(kubeClient, b.Namespace, opts)
	return err
}

func removeContainer(c []api.Container, name string) []api.Container {
	for i, v := range c {
		if v.Name == name {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	return c
}

func removeVolume(volumes []api.Volume, name string) []api.Volume {
	for i, v := range volumes {
		if v.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	return volumes
}

func snapshotRetention(b *rapi.Backup) error {
	cmd := fmt.Sprintf("/restic -r %s forget --%s %d", b.Spec.Destination.Path, b.Spec.BackupRetentionPolicy.Strategy, b.Spec.BackupRetentionPolicy.SnapshotCount)
	if len(b.Spec.BackupRetentionPolicy.RetentionHostname) != 0 {
		cmd = cmd + " --hostname " + b.Spec.BackupRetentionPolicy.RetentionHostname
	}
	if len(b.Spec.BackupRetentionPolicy.RetentionTags) != 0 {
		for _, t := range b.Spec.BackupRetentionPolicy.RetentionTags {
			cmd = cmd + " --tag " + t
		}
	}
	if b.Spec.BackupRetentionPolicy.Prune == true {
		cmd = cmd + " prune "
	}
	_, err := execLocal(cmd)
	return err
}

func execLocal(s string) (string, error) {
	parts := strings.Fields(s)
	head := parts[0]
	parts = parts[1:]
	cmdOut, err := exec.Command(head, parts...).Output()
	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

func getPasswordFromSecret(client *internalclientset.Clientset, secretRef rapi.SecretRef, namespace string) (string, error) {
	secret, err := client.Core().Secrets(namespace).Get(secretRef.Name)
	if err != nil {
		return "", err
	}
	password, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", errors.New("Restic Password not found")
	}
	return string(password), nil
}

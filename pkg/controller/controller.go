package controller

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
	tcs "github.com/appscode/restik/client/clientset"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/restic/restic/src/restic/errors"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
	"reflect"
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

func init() {
	rand.Seed(time.Now().UnixNano())
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
	password, err := getPasswordFromSecret(client, backup.Spec.Destination.RepositorySecretName, backup.Namespace)
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
	event := &api.Event{
		ObjectMeta: api.ObjectMeta{
			Namespace: backup.Namespace,
		},
		InvolvedObject: api.ObjectReference{
			Kind:      backup.Kind,
			Namespace: backup.Namespace,
			Name:      backup.Name,
		},
	}
	c.AddFunc(interval, func() {
		backup, err := extClient.Backup(namespace).Get(tprName)
		if err != nil {
			log.Println(err)
		}
		backup.Status.LastBackupStatus = rapi.StatusUnknown
		password, err := getPasswordFromSecret(client, backup.Spec.Destination.RepositorySecretName, backup.Namespace)
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
		backupOutput, err := execLocal(cmd)
		if err != nil {
			log.Println("Restick backup failed cause ", err)
			backup.Status.LastBackupStatus = rapi.StatusFailed
		}else {
			backup.Status.LastSuccessfullBackup = time.Now()
			backup.Status.LastBackupStatus = rapi.StatusSuccess
		}
		retentionOutput, err := snapshotRetention(backup)
		if err != nil {
			log.Println("Snapshot retention failed cause ", err)
		}
		updateStatusForBackup(backup)
		backup, err = extClient.Backup(backup.Namespace).UpdateStatus(backup)
		if err != nil {
			log.Println(err)
		}
		event.Namespace = backup.Name + "-" + randStringRunes(5)
		event.Message = fmt.Sprintf("Backup : \n %s \n Retention: \n %s", backupOutput, retentionOutput)
		_, err = client.Core().Events(backup.Namespace).Create(event)
		if err != nil {
			log.Println(err)
		}
	})
	wait.Until(func() {}, time.Second, wait.NeverStop)
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
	case ReplicationController:
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
		err = restartPods(kubeClient, b.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = append(replicaset.Spec.Template.Spec.Containers, restikContainer)
		replicaset.Spec.Template.Spec.Volumes = append(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		newReplicaset, err := kubeClient.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		err = restartPods(kubeClient, b.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, restikContainer)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		_, err := kubeClient.Extensions().Deployments(b.Namespace).Update(deployment)
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
		newDaemonset, err := kubeClient.Extensions().DaemonSets(b.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		err = restartPods(kubeClient, b.Namespace, opts)
	case StatefulSet:
		statefulset := &apps.StatefulSet{}
		if err := yaml.Unmarshal(ob, statefulset); err != nil {
			return err
		}
		statefulset.Spec.Template.Spec.Containers = append(statefulset.Spec.Template.Spec.Containers, restikContainer)
		statefulset.Spec.Template.Spec.Volumes = append(statefulset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume)
		_, err := kubeClient.Apps().StatefulSets(b.Namespace).Update(statefulset)
		if err != nil {
			return err
		}
	}
	err = pl.addAnnotation(b)
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
	case ReplicationController:
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
		err = restartPods(kubeClient, b.Namespace, opts)
	case ReplicaSet:
		replicaset := &extensions.ReplicaSet{}
		if err := yaml.Unmarshal(ob, replicaset); err != nil {
			return err
		}
		replicaset.Spec.Template.Spec.Containers = removeContainer(replicaset.Spec.Template.Spec.Containers, ContainerName)
		replicaset.Spec.Template.Spec.Volumes = removeVolume(replicaset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newReplicaset, err := kubeClient.Extensions().ReplicaSets(b.Namespace).Update(replicaset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newReplicaset.Spec.Template.Labels)
		err = restartPods(kubeClient, b.Namespace, opts)
	case DaemonSet:
		daemonset := &extensions.DaemonSet{}
		if err := yaml.Unmarshal(ob, daemonset); err != nil {
			return err
		}
		daemonset.Spec.Template.Spec.Containers = removeContainer(daemonset.Spec.Template.Spec.Containers, ContainerName)
		daemonset.Spec.Template.Spec.Volumes = removeVolume(daemonset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		newDaemonset, err := kubeClient.Extensions().DaemonSets(b.Namespace).Update(daemonset)
		if err != nil {
			return err
		}
		opts.LabelSelector = findSelectors(newDaemonset.Spec.Template.Labels)
		err = restartPods(kubeClient, b.Namespace, opts)
	case Deployment:
		deployment := &extensions.Deployment{}
		if err := yaml.Unmarshal(ob, deployment); err != nil {
			return err
		}
		deployment.Spec.Template.Spec.Containers = removeContainer(deployment.Spec.Template.Spec.Containers, ContainerName)
		deployment.Spec.Template.Spec.Volumes = removeVolume(deployment.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		_, err := kubeClient.Extensions().Deployments(b.Namespace).Update(deployment)
		if err != nil {
			return err
		}
	case StatefulSet:
		statefulset := &apps.StatefulSet{}
		if err := yaml.Unmarshal(ob, statefulset); err != nil {
			return err
		}
		statefulset.Spec.Template.Spec.Containers = removeContainer(statefulset.Spec.Template.Spec.Containers, ContainerName)
		statefulset.Spec.Template.Spec.Volumes = removeVolume(statefulset.Spec.Template.Spec.Volumes, b.Spec.Destination.Volume.Name)
		_, err := kubeClient.Apps().StatefulSets(b.Namespace).Update(statefulset)
		if err != nil {
			return err
		}
	}
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

func snapshotRetention(b *rapi.Backup) (string, error) {
	cmd := fmt.Sprintf("/restic -r %s forget --%s", b.Spec.Destination.Path, b.Spec.RetentionPolicy.Strategy)
	if b.Spec.RetentionPolicy.SnapshotCount != 0 {
		cmd = fmt.Sprintf("%s %d", cmd, b.Spec.RetentionPolicy.SnapshotCount)
	}
	if len(b.Spec.RetentionPolicy.RetainHostname) != 0 {
		cmd = cmd + " --hostname " + b.Spec.RetentionPolicy.RetainHostname
	}
	if len(b.Spec.RetentionPolicy.RetainTags) != 0 {
		for _, t := range b.Spec.RetentionPolicy.RetainTags {
			cmd = cmd + " --tag " + t
		}
	}
	if b.Spec.RetentionPolicy.Prune == true {
		cmd = cmd + " prune "
	}
	output, err := execLocal(cmd)
	return output, err
}

func execLocal(s string) (string, error) {
	parts := strings.Fields(s)
	head := parts[0]
	parts = parts[1:]
	cmdOut, err := exec.Command(head, parts...).Output()
	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

func getPasswordFromSecret(client *internalclientset.Clientset, secretName, namespace string) (string, error) {
	secret, err := client.Core().Secrets(namespace).Get(secretName)
	if err != nil {
		return "", err
	}
	password, ok := secret.Data[Password]
	if !ok {
		return "", errors.New("Restic Password not found")
	}
	return string(password), nil
}

func (pl *Controller) addAnnotation(b *rapi.Backup) error {
	annotation := make(map[string]string)
	annotation["backup.appscode.com/image"] = pl.Image
	b.ObjectMeta.Annotations = annotation
	_, err := pl.Client.Backup(b.Namespace).Update(b)
	return err
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func updateStatusForBackup(backup *rapi.Backup) {
	backup.Status.BackupCount++
	backup.Status.LastBackup = time.Now()
	if reflect.DeepEqual(backup.Status.Created, time.Time{}) {
		backup.Status.Created = time.Now()
	}
}

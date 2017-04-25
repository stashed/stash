package controller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	rapi "github.com/appscode/k8s-addons/api"
	tcs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/log"
	"github.com/appscode/restik/pkg/eventer"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/restic/restic/src/restic/errors"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
	k8serrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	//	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	ExtClient tcs.AppsCodeExtensionInterface
	Client    clientset.Interface
	// sync time to sync the list.
	SyncPeriod time.Duration
	// image of sidecar container
	Image string
}

type cronController struct {
	extClient     tcs.AppsCodeExtensionInterface
	kubeClient    clientset.Interface
	tprName       string
	namespace     string
	crons         *cron.Cron
	backup        *rapi.Backup
	eventRecorder eventer.EventRecorderInterface
}

func New(c *rest.Config, image string) *Controller {
	return &Controller{
		ExtClient:  tcs.NewACExtensionsForConfigOrDie(c),
		Client:     clientset.NewForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
		Image:      image,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Controller) RunAndHold() {
	w.ensureResource()
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
					return
				}
				newObj, ok := new.(*rapi.Backup)
				if !ok {
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
	helper := &cronController{
		extClient:     tcs.NewACExtensionsForConfigOrDie(config),
		kubeClient:    client,
		namespace:     os.Getenv(Namespace),
		tprName:       os.Getenv(TPR),
		crons:         cron.New(),
		eventRecorder: eventer.NewEventRecorder(client, "Restik sidecar Watcher"),
	}

	//get kubeobject
	labelMap := map[string]string{
		BackupObject: "",
	}
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return helper.extClient.Backups(helper.namespace).List(api.ListOptions{
				LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
			})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return helper.extClient.Backups(helper.namespace).Watch(api.ListOptions{
				FieldSelector: labels.SelectorFromSet(labels.Set(labelMap)),
			})
		},
	}
	_, cronController := cache.NewInformer(lw,
		&rapi.Backup{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if b, ok := obj.(*rapi.Backup); ok {
					helper.backup = b
					err = helper.startCronBackupProcedure()
					if err != nil {
						log.Errorln(err)
						return
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
					helper.backup = newObj
					err = helper.startCronBackupProcedure()
					if err != nil {
						log.Errorln(err)
						return
					}
				}
			},
		})
	cronController.Run(wait.NeverStop)
}

func (helper *cronController) startCronBackupProcedure() error {
	backup := helper.backup
	password, err := getPasswordFromSecret(helper.kubeClient, backup.Spec.Destination.RepositorySecretName, backup.Namespace)
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
	for _, v := range helper.crons.Entries() {
		helper.crons.Remove(v.ID)
	}
	interval := backup.Spec.Schedule
	if _, err = cron.Parse(interval); err != nil {
		er := err
		//Reset Wrong Schedule
		helper.backup.Spec.Schedule = ""
		// Create event
		helper.eventRecorder.PushEvent(api.EventTypeWarning, eventer.EventReasonCronExpressionFailed, err.Error(), backup)
		_, err = helper.extClient.Backups(backup.Namespace).Update(backup)
		if err != nil {
			log.Errorln(err)
		}
		return er
	}
	_, err = helper.crons.AddFunc(interval, helper.runCronJob)
	if err != nil {
		return err
	}
	return nil
}

func (helper *cronController) runCronJob() {
	backup := helper.backup
	password, err := getPasswordFromSecret(helper.kubeClient, backup.Spec.Destination.RepositorySecretName, backup.Namespace)
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		log.Errorln(err)
		return
	}
	backupStartTime := unversioned.Now()
	cmd := fmt.Sprintf("/restic -r %s backup %s", backup.Spec.Destination.Path, backup.Spec.Source.Path)
	// add tags if any
	for _, t := range backup.Spec.Tags {
		cmd = cmd + " --tag " + t
	}
	// Force flag
	cmd = cmd + " --" + Force
	// Take Backup
	_, err = execLocal(cmd)
	reason := ""
	if err != nil {
		log.Errorln("Restik backup failed cause ", err)
		reason = eventer.EventReasonBackupFailed
	} else {
		backup.Status.LastSuccessfullBackupTime = &backupStartTime
		reason = eventer.EventReasonBackupSuccess
	}
	backupEndTime := unversioned.Now()
	_, err = snapshotRetention(backup)
	if err != nil {
		log.Errorln("Snapshot retention failed cause ", err)
	}
	backup.Status.BackupCount++
	message := "Backup operation number = " + strconv.Itoa(int(backup.Status.BackupCount))
	helper.eventRecorder.PushEvent(api.EventTypeNormal, reason, message, backup)
	backup.Status.LastBackupTime = &backupStartTime
	if reflect.DeepEqual(backup.Status.FirstBackupTime, time.Time{}) {
		backup.Status.FirstBackupTime = &backupStartTime
	}
	backup.Status.LastBackupDuration = backupEndTime.Sub(backupStartTime.Time).String()
	backup, err = helper.extClient.Backups(backup.Namespace).Update(backup)
	if err != nil {
		log.Errorln(err)
	}
}

func getRestikContainer(b *rapi.Backup, containerImage string) api.Container {
	container := api.Container{
		Name:            ContainerName,
		Image:           containerImage,
		ImagePullPolicy: api.PullAlways,
		Env: []api.EnvVar{
			{
				Name:  Namespace,
				Value: b.Namespace,
			},
			{
				Name:  TPR,
				Value: b.Name,
			},
		},
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

func restartPods(kubeClient clientset.Interface, namespace string, opts api.ListOptions) error {
	pods, err := kubeClient.Core().Pods(namespace).List(opts)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		deleteOpts := &api.DeleteOptions{}
		err = kubeClient.Core().Pods(namespace).Delete(pod.Name, deleteOpts)
		if err != nil {
			errMessage := fmt.Sprintf("Failed to restart pod %s cause %v", pod.Name, err)
			log.Errorln(errMessage)
		}
	}
	return nil
}

func getKubeObject(kubeClient clientset.Interface, namespace string, ls labels.Selector) ([]byte, error, string) {
	rcs, err := kubeClient.Core().ReplicationControllers(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(rcs.Items) != 0 {
		b, err := yaml.Marshal(rcs.Items[0])
		return b, err, ReplicationController
	}

	replicasets, err := kubeClient.Extensions().ReplicaSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(replicasets.Items) != 0 {
		b, err := yaml.Marshal(replicasets.Items[0])
		return b, err, ReplicaSet
	}

	deployments, err := kubeClient.Extensions().Deployments(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(deployments.Items) != 0 {
		b, err := yaml.Marshal(deployments.Items[0])
		return b, err, Deployment
	}

	daemonsets, err := kubeClient.Extensions().DaemonSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(daemonsets.Items) != 0 {
		b, err := yaml.Marshal(daemonsets.Items[0])
		return b, err, DaemonSet
	}

	statefulsets, err := kubeClient.Apps().StatefulSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(statefulsets.Items) != 0 {
		b, err := yaml.Marshal(statefulsets.Items[0])
		return b, err, StatefulSet
	}
	return nil, nil, ""
}

func findSelectors(lb map[string]string) labels.Selector {
	set := labels.Set(lb)
	selectores := labels.SelectorFromSet(set)
	return selectores
}

func (pl *Controller) updateObjectAndStartBackup(b *rapi.Backup) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: b.Name})
	restikContainer := getRestikContainer(b, pl.Image)
	ob, err, _type := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || _type == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
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
	err = pl.addAnnotation(b)
	if err != nil {
		return err
	}
	err = pl.addLabel(b)
	if err != nil {
		return err
	}

}

func (pl *Controller) updateObjectAndStopBackup(b *rapi.Backup) error {
	ls := labels.SelectorFromSet(labels.Set{BackupConfig: b.Name})
	ob, err, _type := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || _type == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
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
	ob, err, _type := getKubeObject(pl.Client, b.Namespace, ls)
	if err != nil {
		return err
	}
	if ob == nil || _type == "" {
		return errors.New(fmt.Sprintf("No object found for backup %s ", b.Name))
	}
	opts := api.ListOptions{}
	switch _type {
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

func (w *Controller) ensureResource() {
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
			log.Fatalln(tpr.Name, "failed to create, causes", err.Error())
		}
	}
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
func updateImageForRestikContainer(c []api.Container, name, image string) []api.Container {
	for i, v := range c {
		if v.Name == name {
			c[i].Image = image
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
	cmd := fmt.Sprintf("/restic -r %s forget", b.Spec.Destination.Path)
	if b.Spec.RetentionPolicy.KeepLastSnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepLast, b.Spec.RetentionPolicy.KeepLastSnapshots)
	}
	if b.Spec.RetentionPolicy.KeepHourlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepHourly, b.Spec.RetentionPolicy.KeepHourlySnapshots)
	}
	if b.Spec.RetentionPolicy.KeepDailySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepDaily, b.Spec.RetentionPolicy.KeepDailySnapshots)
	}
	if b.Spec.RetentionPolicy.KeepWeeklySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepWeekly, b.Spec.RetentionPolicy.KeepWeeklySnapshots)
	}
	if b.Spec.RetentionPolicy.KeepMonthlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepMonthly, b.Spec.RetentionPolicy.KeepMonthlySnapshots)
	}
	if b.Spec.RetentionPolicy.KeepYearlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepYearly, b.Spec.RetentionPolicy.KeepYearlySnapshots)
	}
	if len(b.Spec.RetentionPolicy.KeepTags) != 0 {
		for _, t := range b.Spec.RetentionPolicy.KeepTags {
			cmd = cmd + " --keep-tag " + t
		}
	}
	if len(b.Spec.RetentionPolicy.RetainHostname) != 0 {
		cmd = cmd + " --hostname " + b.Spec.RetentionPolicy.RetainHostname
	}
	if len(b.Spec.RetentionPolicy.RetainTags) != 0 {
		for _, t := range b.Spec.RetentionPolicy.RetainTags {
			cmd = cmd + " --tag " + t
		}
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

func getPasswordFromSecret(client clientset.Interface, secretName, namespace string) (string, error) {
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
	if b.ObjectMeta.Annotations == nil {
		b.ObjectMeta.Annotations = make(map[string]string)
	}
	b.ObjectMeta.Annotations[ImageAnnotation] = pl.Image
	_, err := pl.ExtClient.Backups(b.Namespace).Update(b)
	return err
}

func (pl *Controller) addLabel(b *rapi.Backup) error  {
	if b.ObjectMeta.Labels == nil {
		b.ObjectMeta.Labels = make(map[string]string)
	}

}

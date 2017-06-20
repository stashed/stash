package cron

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/log"
	rapi "github.com/appscode/restik/api"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/analytics"
	"github.com/appscode/restik/pkg/eventer"
	"gopkg.in/robfig/cron.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

const (
	ContainerName      = "restik"
	RestikNamespace    = "RESTIK_NAMESPACE"
	RestikResourceName = "RESTIK_RESOURCE_NAME"

	BackupConfig          = "backup.appscode.com/config"
	RESTIC_PASSWORD       = "RESTIC_PASSWORD"
	ReplicationController = "ReplicationController"
	ReplicaSet            = "ReplicaSet"
	Deployment            = "Deployment"
	DaemonSet             = "DaemonSet"
	StatefulSet           = "StatefulSet"
	Password              = "password"
	ImageAnnotation       = "backup.appscode.com/image"
	Force                 = "force"
)

type controller struct {
	KubeClient   clientset.Interface
	RestikClient rcs.ExtensionInterface

	resourceNamespace string
	resourceName      string
	resource          *rapi.Restik

	crons         *cron.Cron
	eventRecorder record.EventRecorder
}

func NewController(kubeClient clientset.Interface, restikClient rcs.ExtensionInterface, namespace, name string) *controller {
	return &controller{
		KubeClient:        kubeClient,
		RestikClient:      restikClient,
		resourceNamespace: namespace,
		resourceName:      name,
		crons:             cron.New(),
		eventRecorder:     eventer.NewEventRecorder(kubeClient, "restik-crond"),
	}
}

func (c *controller) RunAndHold() {
	c.crons.Start()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.RestikClient.Restiks(c.resourceNamespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.RestikClient.Restiks(c.resourceNamespace).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&rapi.Restik{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*rapi.Restik); ok {
					if r.Name == c.resourceName {
						c.resource = r
						err := c.startCronBackupProcedure()
						if err != nil {
							restikCronFailedToAdd()
							c.eventRecorder.Eventf(
								r,
								apiv1.EventTypeWarning,
								eventer.EventReasonFailedToBackup,
								"Failed to start backup process reason %v", err,
							)
							log.Errorln(err)
						} else {
							restikCronSuccessfullyAdded()
						}
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == c.resourceName {
					c.resource = newObj
					err := c.startCronBackupProcedure()
					if err != nil {
						restikCronFailedToModify()
						c.eventRecorder.Eventf(
							newObj,
							apiv1.EventTypeWarning,
							eventer.EventReasonFailedToBackup,
							"Failed to update backup process reason %v", err,
						)
						log.Errorln(err)
					} else {
						restikCronSuccessfullyModified()
					}
				}
			},
		})
	ctrl.Run(wait.NeverStop)
}

func (cronWatcher *controller) startCronBackupProcedure() error {
	restik := cronWatcher.resource
	password, err := getPasswordFromSecret(cronWatcher.KubeClient, restik.Spec.Destination.RepositorySecretName, restik.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
	}
	repo := restik.Spec.Destination.Path
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
	interval := restik.Spec.Schedule
	if _, err = cron.Parse(interval); err != nil {
		log.Errorln(err)
		cronWatcher.eventRecorder.Event(restik, apiv1.EventTypeWarning, eventer.EventReasonInvalidCronExpression, err.Error())
		//Reset Wrong Schedule
		restik.Spec.Schedule = ""
		_, err = cronWatcher.RestikClient.Restiks(restik.Namespace).Update(restik)
		if err != nil {
			return err
		}
		cronWatcher.eventRecorder.Event(restik, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulCronExpressionReset, "Cron expression reset")
		return nil
	}
	_, err = cronWatcher.crons.AddFunc(interval, func() {
		if err := cronWatcher.runCronJob(); err != nil {
			restikJobFailure()
			cronWatcher.eventRecorder.Event(restik, apiv1.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		} else {
			restikJobSuccess()
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (cronWatcher *controller) runCronJob() error {
	backup := cronWatcher.resource
	password, err := getPasswordFromSecret(cronWatcher.KubeClient, cronWatcher.resource.Spec.Destination.RepositorySecretName, backup.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
	}
	backupStartTime := metav1.Now()
	cmd := fmt.Sprintf("/restic -r %s backup %s", backup.Spec.Destination.Path, backup.Spec.Source.Path)
	// add tags if any
	for _, t := range backup.Spec.Tags {
		cmd = cmd + " --tag " + t
	}
	// Force flag
	cmd = cmd + " --" + Force
	// Take Backup
	var reason string
	errMessage := ""
	_, err = execLocal(cmd)
	if err != nil {
		log.Errorln("Restik backup failed cause ", err)
		errMessage = " ERROR: " + err.Error()
		reason = eventer.EventReasonFailedToBackup
		restikBackupFailure()
	} else {
		backup.Status.LastSuccessfulBackupTime = &backupStartTime
		reason = eventer.EventReasonSuccessfulBackup
		restikBackupSuccess()
	}
	backup.Status.BackupCount++
	message := "Backup operation number = " + strconv.Itoa(int(backup.Status.BackupCount))
	cronWatcher.eventRecorder.Event(backup, apiv1.EventTypeNormal, reason, message+errMessage)
	backupEndTime := metav1.Now()
	_, err = snapshotRetention(backup)
	if err != nil {
		log.Errorln("Snapshot retention failed cause ", err)
		cronWatcher.eventRecorder.Event(backup, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, message+" ERROR: "+err.Error())
	}
	backup.Status.LastBackupTime = &backupStartTime
	if reflect.DeepEqual(backup.Status.FirstBackupTime, time.Time{}) {
		backup.Status.FirstBackupTime = &backupStartTime
	}
	backup.Status.LastBackupDuration = backupEndTime.Sub(backupStartTime.Time).String()
	backup, err = cronWatcher.RestikClient.Restiks(backup.Namespace).Update(backup)
	if err != nil {
		log.Errorln(err)
		cronWatcher.eventRecorder.Event(backup, apiv1.EventTypeNormal, eventer.EventReasonFailedToUpdate, err.Error())
	}
	cronWatcher.resource = backup
	return nil
}

func snapshotRetention(r *rapi.Restik) (string, error) {
	cmd := fmt.Sprintf("/restic -r %s forget", r.Spec.Destination.Path)
	if r.Spec.RetentionPolicy.KeepLastSnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepLast, r.Spec.RetentionPolicy.KeepLastSnapshots)
	}
	if r.Spec.RetentionPolicy.KeepHourlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepHourly, r.Spec.RetentionPolicy.KeepHourlySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepDailySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepDaily, r.Spec.RetentionPolicy.KeepDailySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepWeeklySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepWeekly, r.Spec.RetentionPolicy.KeepWeeklySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepMonthlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepMonthly, r.Spec.RetentionPolicy.KeepMonthlySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepYearlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, rapi.KeepYearly, r.Spec.RetentionPolicy.KeepYearlySnapshots)
	}
	if len(r.Spec.RetentionPolicy.KeepTags) != 0 {
		for _, t := range r.Spec.RetentionPolicy.KeepTags {
			cmd = cmd + " --keep-tag " + t
		}
	}
	if len(r.Spec.RetentionPolicy.RetainHostname) != 0 {
		cmd = cmd + " --hostname " + r.Spec.RetentionPolicy.RetainHostname
	}
	if len(r.Spec.RetentionPolicy.RetainTags) != 0 {
		for _, t := range r.Spec.RetentionPolicy.RetainTags {
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
	secret, err := client.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	password, ok := secret.Data[Password]
	if !ok {
		return "", errors.New("Restic Password not found")
	}
	return string(password), nil
}

func restikCronSuccessfullyAdded() {
	analytics.SendEvent("restic-cron", "added", "success")
}

func restikCronFailedToAdd() {
	analytics.SendEvent("restic-cron", "added", "failure")
}

func restikCronSuccessfullyModified() {
	analytics.SendEvent("restic-cron", "modified", "success")
}

func restikCronFailedToModify() {
	analytics.SendEvent("restic-cron", "modified", "failure")
}

func restikBackupSuccess() {
	analytics.SendEvent("restic-cron", "backup", "success")
}

func restikBackupFailure() {
	analytics.SendEvent("restic-cron", "backup", "failure")
}

func restikJobSuccess() {
	analytics.SendEvent("restic-cron", "job", "success")
}

func restikJobFailure() {
	analytics.SendEvent("restic-cron", "job", "failure")
}

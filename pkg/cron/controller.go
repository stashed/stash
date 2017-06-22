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
	sapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/eventer"
	"gopkg.in/robfig/cron.v2"
	kerr "k8s.io/apimachinery/pkg/api/errors"
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
	RESTIC_PASSWORD = "RESTIC_PASSWORD"
	Password        = "password"
	Force           = "force"
)

type controller struct {
	KubeClient  clientset.Interface
	StashClient scs.ExtensionInterface

	resourceNamespace string
	resourceName      string

	resource        chan *sapi.Restic
	resourceVersion string
	locked          chan struct{}

	scheduler *cron.Cron
	recorder  record.EventRecorder
}

func NewController(kubeClient clientset.Interface, stashClient scs.ExtensionInterface, namespace, name string) *controller {
	return &controller{
		KubeClient:        kubeClient,
		StashClient:       stashClient,
		resourceNamespace: namespace,
		resourceName:      name,
		resource:          make(chan *sapi.Restic),
		recorder:          eventer.NewEventRecorder(kubeClient, "stash-crond"),
	}
}

func (c *controller) RunAndHold() {
	c.scheduler.Start()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.StashClient.Restics(c.resourceNamespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.StashClient.Restics(c.resourceNamespace).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Restic{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.resourceName {
						c.resource <- r
						err := c.configureScheduler()
						if err != nil {
							crondFailedToAdd()
							c.recorder.Eventf(
								r,
								apiv1.EventTypeWarning,
								eventer.EventReasonFailedToBackup,
								"Failed to start backup process reason %v", err,
							)
							log.Errorln(err)
						} else {
							crondSuccessfullyAdded()
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				newObj, ok := new.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == c.resourceName {
					c.resource <- newObj
					err := c.configureScheduler()
					if err != nil {
						crondFailedToModify()
						c.recorder.Eventf(
							newObj,
							apiv1.EventTypeWarning,
							eventer.EventReasonFailedToBackup,
							"Failed to update backup process reason %v", err,
						)
						log.Errorln(err)
					} else {
						crondSuccessfullyModified()
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.resourceName {
						c.scheduler.Stop()
					}
				}
			},
		})
	ctrl.Run(wait.NeverStop)
}

func (c *controller) configureScheduler() error {
	r := <-c.resource
	c.resourceVersion = r.ResourceVersion
	if c.scheduler == nil {
		c.locked = make(chan struct{})
		c.locked <- struct{}{}
		c.scheduler = cron.New()
	}

	password, err := getPasswordFromSecret(c.KubeClient, r.Spec.Destination.RepositorySecretName, r.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
	}
	repo := r.Spec.Destination.Path
	_, err = os.Stat(filepath.Join(repo, "config"))
	if os.IsNotExist(err) {
		if _, err = execLocal(fmt.Sprintf("/restic init --repo %s", repo)); err != nil {
			return err
		}
	}
	// Remove previous jobs
	for _, v := range c.scheduler.Entries() {
		c.scheduler.Remove(v.ID)
	}
	interval := r.Spec.Schedule
	if _, err = cron.Parse(interval); err != nil {
		log.Errorln(err)
		c.recorder.Event(r, apiv1.EventTypeWarning, eventer.EventReasonInvalidCronExpression, err.Error())
		//Reset Wrong Schedule
		r.Spec.Schedule = ""
		_, err = c.StashClient.Restics(r.Namespace).Update(r)
		if err != nil {
			return err
		}
		c.recorder.Event(r, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulCronExpressionReset, "Cron expression reset")
		return nil
	}
	_, err = c.scheduler.AddFunc(interval, func() {
		if err := c.takeBackup(); err != nil {
			stashJobFailure()
			c.recorder.Event(r, apiv1.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		} else {
			stashJobSuccess()
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) takeBackup() error {
	select {
	case <-c.locked:
		log.Infof("Acquired lock for Restic %s@%s", c.resourceName, c.resourceNamespace)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping backup schedule for Restic %s@%s", c.resourceName, c.resourceNamespace)
		return nil
	}

	resource, err := c.StashClient.Restics(c.resourceNamespace).Get(c.resourceName)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if resource.ResourceVersion != c.resourceVersion {
		return fmt.Errorf("Restic %s@%s version %s does not match expected version %s", resource.Name, resource.Namespace, resource.ResourceVersion, c.resourceVersion)
	}

	password, err := getPasswordFromSecret(c.KubeClient, resource.Spec.Destination.RepositorySecretName, resource.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
	}
	backupStartTime := metav1.Now()
	cmd := fmt.Sprintf("/restic -r %s backup %s", resource.Spec.Destination.Path, resource.Spec.Source.Path)
	// add tags if any
	for _, t := range resource.Spec.Tags {
		cmd = cmd + " --tag " + t
	}
	// Force flag
	cmd = cmd + " --" + Force
	// Take Backup
	var reason string
	errMessage := ""
	_, err = execLocal(cmd)
	if err != nil {
		log.Errorln("Stash backup failed cause ", err)
		errMessage = " ERROR: " + err.Error()
		reason = eventer.EventReasonFailedToBackup
		backupFailure()
	} else {
		resource.Status.LastSuccessfulBackupTime = &backupStartTime
		reason = eventer.EventReasonSuccessfulBackup
		backupSuccess()
	}
	resource.Status.BackupCount++
	message := "Backup operation number = " + strconv.Itoa(int(resource.Status.BackupCount))
	c.recorder.Event(resource, apiv1.EventTypeNormal, reason, message+errMessage)
	backupEndTime := metav1.Now()
	_, err = snapshotRetention(resource)
	if err != nil {
		log.Errorln("Snapshot retention failed cause ", err)
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, message+" ERROR: "+err.Error())
	}
	resource.Status.LastBackupTime = &backupStartTime
	if reflect.DeepEqual(resource.Status.FirstBackupTime, time.Time{}) {
		resource.Status.FirstBackupTime = &backupStartTime
	}
	resource.Status.LastBackupDuration = backupEndTime.Sub(backupStartTime.Time).String()
	resource, err = c.StashClient.Restics(resource.Namespace).Update(resource)
	if err != nil {
		log.Errorln(err)
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToUpdate, err.Error())
	}
	return nil
}

func snapshotRetention(r *sapi.Restic) (string, error) {
	cmd := fmt.Sprintf("/restic -r %s forget", r.Spec.Destination.Path)
	if r.Spec.RetentionPolicy.KeepLastSnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepLast, r.Spec.RetentionPolicy.KeepLastSnapshots)
	}
	if r.Spec.RetentionPolicy.KeepHourlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepHourly, r.Spec.RetentionPolicy.KeepHourlySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepDailySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepDaily, r.Spec.RetentionPolicy.KeepDailySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepWeeklySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepWeekly, r.Spec.RetentionPolicy.KeepWeeklySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepMonthlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepMonthly, r.Spec.RetentionPolicy.KeepMonthlySnapshots)
	}
	if r.Spec.RetentionPolicy.KeepYearlySnapshots > 0 {
		cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepYearly, r.Spec.RetentionPolicy.KeepYearlySnapshots)
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

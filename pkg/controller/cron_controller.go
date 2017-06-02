package controller

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
	tcs "github.com/appscode/restik/client/clientset"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

func NewCronController() (*cronController, error) {
	factory := cmdutil.NewFactory(nil)
	config, err := factory.ClientConfig()
	if err != nil {
		return nil, err
	}
	client, err := factory.ClientSet()
	if err != nil {
		return nil, err
	}
	return &cronController{
		extClientset:  tcs.NewForConfigOrDie(config),
		clientset:     client,
		namespace:     os.Getenv(RestikNamespace),
		tprName:       os.Getenv(RestikResourceName),
		crons:         cron.New(),
		eventRecorder: NewEventRecorder(client, "Restik sidecar Watcher"),
	}, nil
}

func (cronWatcher *cronController) RunBackup() error {
	cronWatcher.crons.Start()
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return cronWatcher.extClientset.Restiks(cronWatcher.namespace).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return cronWatcher.extClientset.Restiks(cronWatcher.namespace).Watch(api.ListOptions{})
		},
	}
	_, cronController := cache.NewInformer(lw,
		&rapi.Restik{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*rapi.Restik); ok {
					if r.Name == cronWatcher.tprName {
						cronWatcher.restik = r
						err := cronWatcher.startCronBackupProcedure()
						if err != nil {
							cronWatcher.eventRecorder.Eventf(
								r,
								api.EventTypeWarning,
								EventReasonFailedToBackup,
								"Failed to start backup process reason %v", err,
							)
							log.Errorln(err)
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == cronWatcher.tprName {
					cronWatcher.restik = newObj
					err := cronWatcher.startCronBackupProcedure()
					if err != nil {
						cronWatcher.eventRecorder.Eventf(
							newObj,
							api.EventTypeWarning,
							EventReasonFailedToBackup,
							"Failed to update backup process reason %v", err,
						)
						log.Errorln(err)
					}
				}
			},
		})
	cronController.Run(wait.NeverStop)
	return nil
}

func (cronWatcher *cronController) startCronBackupProcedure() error {
	restik := cronWatcher.restik
	password, err := getPasswordFromSecret(cronWatcher.clientset, restik.Spec.Destination.RepositorySecretName, restik.Namespace)
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
		cronWatcher.eventRecorder.Event(restik, api.EventTypeWarning, EventReasonInvalidCronExpression, err.Error())
		//Reset Wrong Schedule
		restik.Spec.Schedule = ""
		_, err = cronWatcher.extClientset.Restiks(restik.Namespace).Update(restik)
		if err != nil {
			return err
		}
		cronWatcher.eventRecorder.Event(restik, api.EventTypeNormal, EventReasonSuccessfulCronExpressionReset, "Cron expression reset")
		return nil
	}
	_, err = cronWatcher.crons.AddFunc(interval, func() {
		if err := cronWatcher.runCronJob(); err != nil {
			cronWatcher.eventRecorder.Event(restik, api.EventTypeWarning, EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (cronWatcher *cronController) runCronJob() error {
	backup := cronWatcher.restik
	password, err := getPasswordFromSecret(cronWatcher.clientset, cronWatcher.restik.Spec.Destination.RepositorySecretName, backup.Namespace)
	if err != nil {
		return err
	}
	err = os.Setenv(RESTIC_PASSWORD, password)
	if err != nil {
		return err
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
	var reason string
	errMessage := ""
	_, err = execLocal(cmd)
	if err != nil {
		log.Errorln("Restik backup failed cause ", err)
		errMessage = " ERROR: " + err.Error()
		reason = EventReasonFailedToBackup
	} else {
		backup.Status.LastSuccessfulBackupTime = &backupStartTime
		reason = EventReasonSuccessfulBackup
	}
	backup.Status.BackupCount++
	message := "Backup operation number = " + strconv.Itoa(int(backup.Status.BackupCount))
	cronWatcher.eventRecorder.Event(backup, api.EventTypeNormal, reason, message+errMessage)
	backupEndTime := unversioned.Now()
	_, err = snapshotRetention(backup)
	if err != nil {
		log.Errorln("Snapshot retention failed cause ", err)
		cronWatcher.eventRecorder.Event(backup, api.EventTypeNormal, EventReasonFailedToRetention, message+" ERROR: "+err.Error())
	}
	backup.Status.LastBackupTime = &backupStartTime
	if reflect.DeepEqual(backup.Status.FirstBackupTime, time.Time{}) {
		backup.Status.FirstBackupTime = &backupStartTime
	}
	backup.Status.LastBackupDuration = backupEndTime.Sub(backupStartTime.Time).String()
	backup, err = cronWatcher.extClientset.Restiks(backup.Namespace).Update(backup)
	if err != nil {
		log.Errorln(err)
		cronWatcher.eventRecorder.Event(backup, api.EventTypeNormal, EventReasonFailedToUpdate, err.Error())
	}
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

package scheduler

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
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

	prefixHostname bool
	scratchDir     string

	resource        chan *sapi.Restic
	resourceVersion string
	locked          chan struct{}

	cron     *cron.Cron
	recorder record.EventRecorder
}

func NewController(kubeClient clientset.Interface, stashClient scs.ExtensionInterface, namespace, name string, prefixHostname bool, scratchDir string) *controller {
	return &controller{
		KubeClient:        kubeClient,
		StashClient:       stashClient,
		resourceNamespace: namespace,
		resourceName:      name,
		prefixHostname:    prefixHostname,
		scratchDir:        scratchDir,
		resource:          make(chan *sapi.Restic),
		recorder:          eventer.NewEventRecorder(kubeClient, "stash-scheduler"),
	}
}

// Init and/or connect to repo
func (c *controller) InitRepo() error {
	resource, err := c.StashClient.Restics(c.resourceNamespace).Get(c.resourceName)
	if err != nil {
		return err
	}
	data, err := c.getRepositorySecret(resource.Spec.Backend.RepositorySecretName, resource.Namespace)
	if err != nil {
		return err
	}

	err = os.Setenv(RESTIC_PASSWORD, string(data[Password]))
	if err != nil {
		return err
	}
	repo := resource.Spec.Backend.Local.Path
	_, err = os.Stat(filepath.Join(repo, "config"))
	if os.IsNotExist(err) {
		if _, err = execLocal(fmt.Sprintf("/restic init --repo %s", repo)); err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) RunAndHold() {
	c.cron.Start()

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
							schedulerFailedToAdd()
							c.recorder.Eventf(
								r,
								apiv1.EventTypeWarning,
								eventer.EventReasonFailedToBackup,
								"Failed to start backup process reason %v", err,
							)
							log.Errorln(err)
						} else {
							schedulerSuccessfullyAdded()
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
						schedulerFailedToModify()
						c.recorder.Eventf(
							newObj,
							apiv1.EventTypeWarning,
							eventer.EventReasonFailedToBackup,
							"Failed to update backup process reason %v", err,
						)
						log.Errorln(err)
					} else {
						schedulerSuccessfullyModified()
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.resourceName {
						c.cron.Stop()
					}
				}
			},
		})
	ctrl.Run(wait.NeverStop)
}

func (c *controller) configureScheduler() error {
	r := <-c.resource
	c.resourceVersion = r.ResourceVersion
	if c.cron == nil {
		c.locked = make(chan struct{})
		c.locked <- struct{}{}
		c.cron = cron.New()
	}

	err := restic.ExportEnvVars(c.KubeClient, r, c.prefixHostname, c.scratchDir)
	if err != nil {
		return err
	}

	//password, err := getPasswordFromSecret(c.KubeClient, r.Spec.Backend.RepositorySecretName, r.Namespace)
	//if err != nil {
	//	return err
	//}
	//err = os.Setenv(RESTIC_PASSWORD, password)
	//if err != nil {
	//	return err
	//}
	//repo := r.Spec.Backend.Path
	//_, err = os.Stat(filepath.Join(repo, "config"))
	//if os.IsNotExist(err) {
	//	if _, err = execLocal(fmt.Sprintf("/restic init --repo %s", repo)); err != nil {
	//		return err
	//	}
	//}

	// Remove previous jobs
	for _, v := range c.cron.Entries() {
		c.cron.Remove(v.ID)
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
	_, err = c.cron.AddFunc(interval, func() {
		if err := c.runOnce(); err != nil {
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

func (c *controller) runOnce() error {
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

	err = c.runBackup(resource)
	if err != nil {
		log.Errorln("Backup operation failed for Reestic %s@%s due to %s", resource.Name, resource.Namespace, err)
		backupFailure()
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToBackup, " ERROR: "+err.Error())
	} else {
		backupSuccess()
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backup completed successfully.")
	}

	err = forgetSnapshots(resource)
	if err != nil {
		log.Errorln("Failed to forget old snapshots for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, " ERROR: "+err.Error())
	}

	return nil
}

func (c *controller) runBackup(resource *sapi.Restic) error {
	startTime := metav1.Now()
	for _, fg := range resource.Spec.FileGroups {
		cmd := fmt.Sprintf("/restic backup %s", fg.Path)
		// add tags if any
		for _, tag := range fg.Tags {
			cmd = cmd + " --tag " + tag
		}
		// Force flag
		cmd = cmd + " --force"

		_, err := execLocal(cmd)
		if err != nil {
			return err
		}
	}
	endTime := metav1.Now()

	resource.Status.BackupCount++
	resource.Status.LastBackupTime = &startTime
	if resource.Status.FirstBackupTime == nil {
		resource.Status.FirstBackupTime = &startTime
	}
	resource.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
	_, err := c.StashClient.Restics(resource.Namespace).Update(resource)
	if err != nil {
		log.Errorf("Failed to update status for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
	}
	return nil
}

func forgetSnapshots(r *sapi.Restic) error {
	for _, fg := range r.Spec.FileGroups {
		cmd := "/restic forget"
		if fg.RetentionPolicy.KeepLastSnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepLast, fg.RetentionPolicy.KeepLastSnapshots)
		}
		if fg.RetentionPolicy.KeepHourlySnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepHourly, fg.RetentionPolicy.KeepHourlySnapshots)
		}
		if fg.RetentionPolicy.KeepDailySnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepDaily, fg.RetentionPolicy.KeepDailySnapshots)
		}
		if fg.RetentionPolicy.KeepWeeklySnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepWeekly, fg.RetentionPolicy.KeepWeeklySnapshots)
		}
		if fg.RetentionPolicy.KeepMonthlySnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepMonthly, fg.RetentionPolicy.KeepMonthlySnapshots)
		}
		if fg.RetentionPolicy.KeepYearlySnapshots > 0 {
			cmd = fmt.Sprintf("%s --%s %d", cmd, sapi.KeepYearly, fg.RetentionPolicy.KeepYearlySnapshots)
		}
		if len(fg.RetentionPolicy.KeepTags) != 0 {
			for _, t := range fg.RetentionPolicy.KeepTags {
				cmd = cmd + " --keep-tag " + t
			}
		}
		// Debug
		//if len(fg.RetentionPolicy.RetainHostname) != 0 {
		//	cmd = cmd + " --hostname " + fg.RetentionPolicy.RetainHostname
		//}
		for _, t := range fg.Tags {
			cmd = cmd + " --tag " + t
		}
		_, err := execLocal(cmd)
		if err != nil {
			return err
		}
	}
	return nil
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

func (c *controller) getRepositorySecret(secretName, namespace string) (map[string]string, error) {
	secret, err := c.KubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	data := make(map[string]string)
	for k, v := range secret.Data {
		data[k] = string(v)
	}
	return data, nil
}

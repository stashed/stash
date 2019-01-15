package backup

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	rbac_util "github.com/appscode/kutil/rbac/v1"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	stash_listers "github.com/appscode/stash/client/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	cron "gopkg.in/robfig/cron.v2"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
)

type Options struct {
	Workload         api.LocalTypedReference
	Namespace        string
	ResticName       string
	ScratchDir       string
	PushgatewayURL   string
	NodeName         string
	PodName          string
	SmartPrefix      string
	SnapshotHostname string
	PodLabelsPath    string
	QPS              float64
	Burst            int
	ResyncPeriod     time.Duration
	MaxNumRequeues   int
	RunViaCron       bool
	DockerRegistry   string // image registry for check job
	ImageTag         string // image tag for check job
	EnableRBAC       bool   // rbac for check job
	NumThreads       int
}

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.Interface
	opt         Options
	locked      chan struct{}
	resticCLI   *cli.ResticWrapper
	cron        *cron.Cron
	recorder    record.EventRecorder

	stashInformerFactory stashinformers.SharedInformerFactory

	// Restic
	rQueue    *queue.Worker
	rInformer cache.SharedIndexInformer
	rLister   stash_listers.ResticLister
}

const (
	CheckRole            = "stash-check"
	BackupEventComponent = "stash-backup"
)

func New(k8sClient kubernetes.Interface, stashClient cs.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		opt:         opt,
		cron:        cron.New(),
		locked:      make(chan struct{}, 1),
		resticCLI:   cli.New(opt.ScratchDir, true, opt.SnapshotHostname),
		recorder:    eventer.NewEventRecorder(k8sClient, BackupEventComponent),
		stashInformerFactory: stashinformers.NewFilteredSharedInformerFactory(
			stashClient,
			opt.ResyncPeriod,
			opt.Namespace,
			// BUG!!! In 1.8.x, field selectors can't be used with CRDs
			// ref: https://github.com/appscode/voyager/issues/889
			//func(options *metav1.ListOptions) {
			//	options.FieldSelector = fields.OneTermEqualSelector("metadata.name", opt.ResticName).String()
			//},
			nil,
		),
	}
}

func (c *Controller) Backup() error {
	restic, repository, err := c.setup()
	if err != nil {
		err = fmt.Errorf("failed to setup backup. Error: %v", err)
		if restic != nil {
			ref, rerr := reference.GetReference(scheme.Scheme, restic)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedSetup,
					err.Error(),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", restic.Kind, restic.Name, rerr)
			}
		}
		return err
	}

	if err = c.runResticBackup(restic, repository); err != nil {
		return fmt.Errorf("failed to run backup, reason: %s", err)
	}

	// create check job
	image := docker.Docker{
		Registry: c.opt.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.opt.ImageTag,
	}

	job := util.NewCheckJob(restic, c.opt.SnapshotHostname, c.opt.SmartPrefix, image)

	// check if check job exists
	if _, err = c.k8sClient.BatchV1().Jobs(restic.Namespace).Get(job.Name, metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedCronJob,
				err.Error(),
			)
		} else {
			log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
		return err
	}
	if errors.IsNotFound(err) {
		if c.opt.EnableRBAC {
			job.Spec.Template.Spec.ServiceAccountName = job.Name
		}
		if job, err = c.k8sClient.BatchV1().Jobs(restic.Namespace).Create(job); err != nil {
			err = fmt.Errorf("failed to get check job, reason: %s", err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedCronJob,
					err.Error(),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return err
		}

		// create service-account and role-binding
		if c.opt.EnableRBAC {
			ref, err := reference.GetReference(scheme.Scheme, job)
			if err != nil {
				return err
			}
			if err = c.ensureCheckRBAC(ref); err != nil {
				return fmt.Errorf("error ensuring rbac for check job %s, reason: %s", job.Name, err)
			}
		}

		log.Infoln("Created check job:", job.Name)
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeNormal,
				eventer.EventReasonCheckJobCreated,
				fmt.Sprintf("Created check job: %s", job.Name),
			)
		} else {
			log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
	} else {
		log.Infoln("Check job already exists, skipping creation:", job.Name)
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeNormal,
				eventer.EventReasonCheckJobCreated,
				fmt.Sprintf("Check job already exists, skipping creation: %s", job.Name),
			)
		} else {
			log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
	}
	return nil
}

// Init and/or connect to repo
func (c *Controller) setup() (*api.Restic, *api.Repository, error) {
	// setup scratch-dir
	if err := os.MkdirAll(c.opt.ScratchDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create scratch dir: %s", err)
	}
	if err := ioutil.WriteFile(c.opt.ScratchDir+"/.stash", []byte("test"), 644); err != nil {
		return nil, nil, fmt.Errorf("no write access in scratch dir: %s", err)
	}

	// check restic
	restic, err := c.stashClient.StashV1alpha1().Restics(c.opt.Namespace).Get(c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	log.Infof("Found restic %s", restic.Name)
	if err := restic.IsValid(); err != nil {
		return restic, nil, err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(restic.Namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return restic, nil, err
	}
	log.Infof("Found repository secret %s", secret.Name)

	// setup restic-cli
	prefix := ""
	if prefix, err = c.resticCLI.SetupEnv(restic.Spec.Backend, secret, c.opt.SmartPrefix); err != nil {
		return restic, nil, err
	}
	if err = c.resticCLI.InitRepositoryIfAbsent(); err != nil {
		return restic, nil, err
	}
	repository, err := c.createRepositoryCrdIfNotExist(restic, prefix)
	if err != nil {
		return restic, nil, err
	}
	return restic, repository, nil
}

func (c *Controller) runResticBackup(restic *api.Restic, repository *api.Repository) (err error) {
	if restic.Spec.Paused == true {
		log.Infoln("skipped logging since restic is paused.")
		return nil
	}
	startTime := metav1.Now()
	var (
		restic_session_success = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "success",
			Help:      "Indicates if session was successfully completed",
		})
		restic_session_fail = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "fail",
			Help:      "Indicates if session failed",
		})
		restic_session_duration_seconds_total = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds_total",
			Help:      "Total seconds taken to complete restic session",
		})
		restic_session_duration_seconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds",
			Help:      "Total seconds taken to complete restic session",
		}, []string{"filegroup", "op"})
	)

	defer func() {
		endTime := metav1.Now()
		if c.opt.PushgatewayURL != "" {
			if err != nil {
				restic_session_success.Set(0)
				restic_session_fail.Set(1)
			} else {
				restic_session_success.Set(1)
				restic_session_fail.Set(0)
			}
			restic_session_duration_seconds_total.Set(endTime.Sub(startTime.Time).Seconds())

			push.Collectors(c.JobName(restic),
				c.GroupingKeys(restic),
				c.opt.PushgatewayURL,
				restic_session_success,
				restic_session_fail,
				restic_session_duration_seconds_total,
				restic_session_duration_seconds)
		}
		if err == nil {
			stash_util.UpdateRepositoryStatus(c.stashClient.StashV1alpha1(), repository, func(in *api.RepositoryStatus) *api.RepositoryStatus {
				in.BackupCount++
				in.LastBackupTime = &startTime
				if in.FirstBackupTime == nil {
					in.FirstBackupTime = &startTime
				}
				in.LastBackupDuration = endTime.Sub(startTime.Time).String()
				return in
			}, apis.EnableStatusSubresource)
		}
	}()

	for _, fg := range restic.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, restic, fg, backupOpMetric)
		if err != nil {
			log.Errorf("Backup failed for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToBackup,
					fmt.Sprintf("Backup failed, reason: %s", err),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return
		} else {
			hostname, _ := os.Hostname()
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeNormal,
					eventer.EventReasonSuccessfulBackup,
					fmt.Sprintf("Backed up pod: %s, path: %s", hostname, fg.Path),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, restic, fg, forgetOpMetric)
		if err != nil {
			log.Errorf("Failed to forget old snapshots for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToRetention,
					fmt.Sprintf("Failed to forget old snapshots, reason: %s", err),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return
		}
	}
	return
}

func (c *Controller) measure(f func(*api.Restic, api.FileGroup) error, restic *api.Restic, fg api.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Now().Sub(startTime).Seconds())
	}()
	err = f(restic, fg)
	return
}

// use sidecar-cluster-role, service-account and role-binding name same as job name
// set job as owner of service-account and role-binding
func (c *Controller) ensureCheckRBAC(restic *core.ObjectReference) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      restic.Name,
		Namespace: restic.Namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(c.k8sClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, restic)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"
		return in
	})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(c.k8sClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, restic)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     controller.SidecarClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		}
		return in
	})
	return err
}

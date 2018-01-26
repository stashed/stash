package backup

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	rbac_util "github.com/appscode/kutil/rbac/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/typed/stash/v1alpha1/util"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/robfig/cron.v2"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/client-go/util/workqueue"
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
	ResyncPeriod     time.Duration
	MaxNumRequeues   int
	RunViaCron       bool
	DockerRegistry   string // image registry for check job
	ImageTag         string // image tag for check job
	EnableRBAC       bool   // rbac for check job
}

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	opt         Options
	locked      chan struct{}
	resticCLI   *cli.ResticWrapper
	cron        *cron.Cron
	recorder    record.EventRecorder

	// Restic
	rQueue    workqueue.RateLimitingInterface
	rIndexer  cache.Indexer
	rInformer cache.Controller
	rLister   stash_listers.ResticLister
}

const (
	CheckRole            = "stash-check"
	BackupEventComponent = "stash-backup"
)

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		opt:         opt,
		cron:        cron.New(),
		locked:      make(chan struct{}, 1),
		resticCLI:   cli.New(opt.ScratchDir, true, opt.SnapshotHostname),
		recorder:    eventer.NewEventRecorder(k8sClient, BackupEventComponent),
	}
}

func (c *Controller) Backup() error {
	resource, err := c.setup()
	if err != nil {
		return fmt.Errorf("failed to setup backup: %s", err)
	}

	if err := c.runResticBackup(resource); err != nil {
		eventer.CreateEventWithLog(
			c.k8sClient,
			BackupEventComponent,
			resource.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedCronJob,
			fmt.Sprintf("Failed to run backup, reason: %s", err),
		)
		return fmt.Errorf("failed to run backup, reason: %s", err)
	}

	// create check job
	image := docker.Docker{
		Registry: c.opt.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.opt.ImageTag,
	}

	job := util.NewCheckJob(resource, c.opt.SnapshotHostname, c.opt.SmartPrefix, image)
	if c.opt.EnableRBAC {
		job.Spec.Template.Spec.ServiceAccountName = job.Name
	}
	if job, err = c.k8sClient.BatchV1().Jobs(resource.Namespace).Create(job); err != nil {
		err = fmt.Errorf("failed to create check job, reason: %s", err)
		eventer.CreateEventWithLog(
			c.k8sClient,
			BackupEventComponent,
			resource.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedCronJob,
			err.Error(),
		)
		return err
	}

	// create service-account and role-binding
	if c.opt.EnableRBAC {
		ref, err := reference.GetReference(scheme.Scheme, job)
		if err != nil {
			return err
		}
		if err = c.ensureCheckRBAC(ref); err != nil {
			return fmt.Errorf("error ensuring rbac for check job %s, reason: %s\n", job.Name, err)
		}
	}

	log.Infoln("Created check job:", job.Name)
	eventer.CreateEventWithLog(
		c.k8sClient,
		BackupEventComponent,
		resource.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonCheckJobCreated,
		fmt.Sprintf("Created check job: %s", job.Name),
	)
	return nil
}

// Init and/or connect to repo
func (c *Controller) setup() (*api.Restic, error) {
	// setup scratch-dir
	if err := os.MkdirAll(c.opt.ScratchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create scratch dir: %s", err)
	}
	if err := ioutil.WriteFile(c.opt.ScratchDir+"/.stash", []byte("test"), 644); err != nil {
		return nil, fmt.Errorf("no write access in scratch dir: %s", err)
	}

	// check resource
	resource, err := c.stashClient.Restics(c.opt.Namespace).Get(c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	log.Infof("Found restic %s\n", resource.Name)
	if err := resource.IsValid(); err != nil {
		return nil, err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	log.Infof("Found repository secret %s\n", secret.Name)

	// setup restic-cli
	if err = c.resticCLI.SetupEnv(resource.Spec.Backend, secret, c.opt.SmartPrefix); err != nil {
		return nil, err
	}
	if err = c.resticCLI.InitRepositoryIfAbsent(); err != nil {
		return nil, err
	}

	return resource, nil
}

func (c *Controller) runResticBackup(resource *api.Restic) (err error) {
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

			push.Collectors(c.JobName(resource),
				c.GroupingKeys(resource),
				c.opt.PushgatewayURL,
				restic_session_success,
				restic_session_fail,
				restic_session_duration_seconds_total,
				restic_session_duration_seconds)
		}

		stash_util.PatchRestic(c.stashClient, resource, func(in *api.Restic) *api.Restic {
			in.Status.BackupCount++
			in.Status.LastBackupTime = &startTime
			if in.Status.FirstBackupTime == nil {
				in.Status.FirstBackupTime = &startTime
			}
			in.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
			return in
		})
	}()

	for _, fg := range resource.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, resource, fg, backupOpMetric)
		if err != nil {
			log.Errorf("Backup operation failed for Restic %s/%s due to %s\n", resource.Namespace, resource.Name, err)
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				resource.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToBackup,
				fmt.Sprintf("Backup operation failed for Restic %s/%s due to %s", resource.Namespace, resource.Name, err),
			)
			return
		} else {
			hostname, _ := os.Hostname()
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				resource.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonSuccessfulBackup,
				fmt.Sprintf("Backed up pod: %s, path: %s", hostname, fg.Path),
			)
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, resource, fg, forgetOpMetric)
		if err != nil {
			log.Errorf("Failed to forget old snapshots for Restic %s/%s due to %s\n", resource.Namespace, resource.Name, err)
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				resource.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToRetention,
				fmt.Sprintf("Failed to forget old snapshots for Restic %s/%s due to %s", resource.Namespace, resource.Name, err),
			)
			return
		}
	}
	return
}

func (c *Controller) measure(f func(*api.Restic, api.FileGroup) error, resource *api.Restic, fg api.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Now().Sub(startTime).Seconds())
	}()
	err = f(resource, fg)
	return
}

// use sidecar-cluster-role, service-account and role-binding name same as job name
// set job as owner of service-account and role-binding
func (c *Controller) ensureCheckRBAC(resource *core.ObjectReference) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(c.k8sClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		in.ObjectMeta = util.EnsureOwnerReference(in.ObjectMeta, resource)
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
		in.ObjectMeta = util.EnsureOwnerReference(in.ObjectMeta, resource)

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

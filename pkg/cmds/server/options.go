/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/controller"

	"github.com/spf13/pflag"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"kmodules.xyz/client-go/discovery"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
)

type ExtraOptions struct {
	LicenseFile             string
	StashImage              string
	StashImageTag           string
	DockerRegistry          string
	ImagePullSecrets        []string
	MaxNumRequeues          int
	NumThreads              int
	ScratchDir              string
	QPS                     float64
	Burst                   int
	ResyncPeriod            time.Duration
	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool
	CronJobPSPNames         []string
	BackupJobPSPNames       []string
	RestoreJobPSPNames      []string
}

func NewExtraOptions() *ExtraOptions {
	return &ExtraOptions{
		DockerRegistry: docker.ACRegistry,
		StashImage:     docker.ImageStash,
		StashImageTag:  "",
		MaxNumRequeues: 5,
		NumThreads:     2,
		ScratchDir:     "/tmp",
		QPS:            100,
		Burst:          100,
		ResyncPeriod:   10 * time.Minute,
	}
}

func (s *ExtraOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.ScratchDir, "scratch-dir", s.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	fs.StringVar(&s.StashImage, "image", s.StashImage, "Image for sidecar, init-container, check-job and recovery-job")
	fs.StringVar(&s.StashImageTag, "image-tag", s.StashImageTag, "Image tag for sidecar, init-container, check-job and recovery-job")
	fs.StringVar(&s.DockerRegistry, "docker-registry", s.DockerRegistry, "Docker image registry for sidecar, init-container, check-job, recovery-job and kubectl-job")
	fs.StringSliceVar(&s.ImagePullSecrets, "image-pull-secrets", s.ImagePullSecrets, "List of image pull secrets for pulling image from private registries")
	fs.StringVar(&s.LicenseFile, "license-file", s.LicenseFile, "Path to license file")

	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	fs.BoolVar(&s.EnableMutatingWebhook, "enable-mutating-webhook", s.EnableMutatingWebhook, "If true, enables mutating webhooks for KubeDB CRDs.")
	fs.BoolVar(&s.EnableValidatingWebhook, "enable-validating-webhook", s.EnableValidatingWebhook, "If true, enables validating webhooks for KubeDB CRDs.")

	fs.StringSliceVar(&s.CronJobPSPNames, "cron-job-psp", s.CronJobPSPNames, "Name of the PSPs for backup triggering CronJob. Use comma to separate multiple PSP names.")
	fs.StringSliceVar(&s.BackupJobPSPNames, "backup-job-psp", s.BackupJobPSPNames, "Name of the PSPs for backup job. Use comma to separate multiple PSP names.")
	fs.StringSliceVar(&s.RestoreJobPSPNames, "restore-job-psp", s.RestoreJobPSPNames, "Name of the PSPs for restore job. Use comma to separate multiple PSP names.")
}

func (s *ExtraOptions) ApplyTo(cfg *controller.Config) error {
	var err error

	cfg.LicenseFile = s.LicenseFile
	cfg.StashImage = s.StashImage
	cfg.StashImageTag = s.StashImageTag
	cfg.DockerRegistry = s.DockerRegistry
	cfg.ImagePullSecrets = s.ImagePullSecrets
	cfg.MaxNumRequeues = s.MaxNumRequeues
	cfg.NumThreads = s.NumThreads
	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst
	cfg.EnableMutatingWebhook = s.EnableMutatingWebhook
	cfg.EnableValidatingWebhook = s.EnableValidatingWebhook

	cfg.CronJobPSPNames = s.CronJobPSPNames
	cfg.BackupJobPSPNames = s.BackupJobPSPNames
	cfg.RestoreJobPSPNames = s.RestoreJobPSPNames

	if cfg.KubeClient, err = kubernetes.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.StashClient, err = cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.CRDClient, err = crd_cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.AppCatalogClient, err = appcatalog_cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}

	// if cluster has OpenShift DeploymentConfig then generate OcClient
	if discovery.IsPreferredAPIResource(cfg.KubeClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
		if cfg.OcClient, err = oc_cs.NewForConfig(cfg.ClientConfig); err != nil {
			return err
		}
	}

	return nil
}

func (s *ExtraOptions) Validate() []error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.StashImageTag == "" {
		errs = append(errs, fmt.Errorf("--image-tag must be specified"))
	}
	return errs
}

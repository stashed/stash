package server

import (
	"flag"
	"time"

	stringz "github.com/appscode/go/strings"
	v "github.com/appscode/go/version"
	"github.com/appscode/stash/apis"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/spf13/pflag"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
)

type ExtraOptions struct {
	EnableRBAC              bool
	StashImageTag           string
	DockerRegistry          string
	MaxNumRequeues          int
	NumThreads              int
	ScratchDir              string
	QPS                     float64
	Burst                   int
	ResyncPeriod            time.Duration
	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool
}

func NewExtraOptions() *ExtraOptions {
	return &ExtraOptions{
		DockerRegistry: docker.ACRegistry,
		StashImageTag:  stringz.Val(v.Version.Version, "canary"),
		MaxNumRequeues: 5,
		NumThreads:     2,
		ScratchDir:     "/tmp",
		QPS:            100,
		Burst:          100,
		ResyncPeriod:   10 * time.Minute,
	}
}

func (s *ExtraOptions) AddGoFlags(fs *flag.FlagSet) {
	fs.BoolVar(&s.EnableRBAC, "rbac", s.EnableRBAC, "Enable RBAC for operator")
	fs.StringVar(&s.ScratchDir, "scratch-dir", s.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	fs.StringVar(&s.StashImageTag, "image-tag", s.StashImageTag, "Image tag for sidecar, init-container, check-job and recovery-job")
	fs.StringVar(&s.DockerRegistry, "docker-registry", s.DockerRegistry, "Docker image registry for sidecar, init-container, check-job, recovery-job and kubectl-job")

	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	fs.BoolVar(&s.EnableMutatingWebhook, "enable-mutating-webhook", s.EnableMutatingWebhook, "If true, enables mutating webhooks for KubeDB CRDs.")
	fs.BoolVar(&s.EnableValidatingWebhook, "enable-validating-webhook", s.EnableValidatingWebhook, "If true, enables validating webhooks for KubeDB CRDs.")
	fs.BoolVar(&apis.EnableStatusSubresource, "enable-status-subresource", apis.EnableStatusSubresource, "If true, uses sub resource for KubeDB crds.")

}

func (s *ExtraOptions) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("stash", flag.ExitOnError)
	s.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (s *ExtraOptions) ApplyTo(cfg *controller.Config) error {
	var err error

	cfg.EnableRBAC = s.EnableRBAC
	cfg.StashImageTag = s.StashImageTag
	cfg.DockerRegistry = s.DockerRegistry
	cfg.MaxNumRequeues = s.MaxNumRequeues
	cfg.NumThreads = s.NumThreads
	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst
	cfg.EnableMutatingWebhook = s.EnableMutatingWebhook
	cfg.EnableValidatingWebhook = s.EnableValidatingWebhook

	if cfg.KubeClient, err = kubernetes.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.StashClient, err = cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.CRDClient, err = crd_cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	return nil
}

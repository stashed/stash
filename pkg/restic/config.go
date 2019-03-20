package restic

import (
	"github.com/appscode/stash/apis/stash/v1alpha1"
	shell "github.com/codeskyblue/go-sh"
)

const (
	DefaultOutputFileName = "output.json"
	DefaultScratchDir     = "/tmp/restic/scratch"
)

type ResticWrapper struct {
	sh     *shell.Session
	config SetupOptions
}

type BackupOptions struct {
	Host            string
	BackupDirs      []string
	RetentionPolicy v1alpha1.RetentionPolicy
}

type RestoreOptions struct {
	Host        string
	RestoreDirs []string
	Snapshots   []string // when Snapshots are specified Host and RestoreDirs will not be used
}

type SetupOptions struct {
	Provider    string
	Bucket      string
	Endpoint    string
	Path        string
	SecretDir   string
	CacertFile  string
	ScratchDir  string
	EnableCache bool
}

type MetricsOptions struct {
	Enabled        bool
	PushgatewayURL string
	MetricFileDir  string
	Labels         []string
	JobName        string
}

func NewResticWrapper(options SetupOptions) (*ResticWrapper, error) {
	wrapper := &ResticWrapper{
		sh:     shell.NewSession(),
		config: options,
	}
	wrapper.sh.SetDir(wrapper.config.ScratchDir)
	wrapper.sh.ShowCMD = true

	// Setup restic environments
	err := wrapper.setupEnv()
	if err != nil {
		return nil, err
	}
	return wrapper, nil
}

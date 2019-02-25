package restic

import (
	v1beta1_api "github.com/appscode/stash/apis/stash/v1beta1"
	shell "github.com/codeskyblue/go-sh"
)

type ResticWrapper struct {
	sh     *shell.Session
	config SetupOptions
}

type BackupOptions struct {
	BackupDirs []string
	Cleanup    CleanupOptions
}

type RestoreOptions struct {
	Rules []v1beta1_api.Rule
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
	Hostname    string
}

type CleanupOptions struct {
	RetentionPolicyName string
	RetentionValue      string
	Prune               bool
	DryRun              bool
}

type RetentionPolicy struct {
	Policy string
	Value  string
	Prune  bool
	DryRun bool
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

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

type Command struct {
	Name string
	Args []interface{}
}

// if StdinPipeCommand is specified, BackupDirs will not be used
type BackupOptions struct {
	Host             string
	BackupDirs       []string
	StdinPipeCommand Command
	StdinFileName    string // default "stdin"
	RetentionPolicy  v1alpha1.RetentionPolicy
}

type RestoreOptions struct {
	Host        string
	SourceHost  string
	RestoreDirs []string
	Snapshots   []string // when Snapshots are specified SourceHost and RestoreDirs will not be used
}

type DumpOptions struct {
	Host              string
	Snapshot          string // default "latest"
	Path              string
	FileName          string // default "stdin"
	StdoutPipeCommand Command
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

func (w *ResticWrapper) SetEnv(key, value string) {
	if w.sh != nil {
		w.sh.SetEnv(key, value)
	}
}

func (w *ResticWrapper) HideCMD() {
	if w.sh != nil {
		w.sh.ShowCMD = false
	}
}

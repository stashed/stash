package restic

import (
	shell "github.com/codeskyblue/go-sh"
)

type ResticWrapper struct {
	sh          *shell.Session
	scratchDir  string
	enableCache bool
	hostname    string
	cacertFile  string
	secretDir   string
}

type BackupOptions struct {
	ScratchDir      string
	EnableCache     bool
	Hostname        string
	OutputDir       string
	Provider        string
	Bucket          string
	Endpoint        string
	Path            string
	SecretDir       string
	RetentionPolicy RetentionPolicy
}

type RetentionPolicy struct {
	Policy string
	Value  string
	Prune  bool
	DryRun bool
}

func NewResticWrapper(scratchDir string, enableCache bool, hostname string) *ResticWrapper {
	ctrl := &ResticWrapper{
		sh:          shell.NewSession(),
		scratchDir:  scratchDir,
		enableCache: enableCache,
		hostname:    hostname,
	}
	ctrl.sh.SetDir(scratchDir)
	ctrl.sh.ShowCMD = true
	return ctrl
}

package util

import (
	go_str "github.com/appscode/go/strings"
	api_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	api "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/restic"
)

// options that don't come from repository, backup-config, backup-session, restore-session
type ExtraOptions struct {
	Host        string
	SecretDir   string
	CacertFile  string
	ScratchDir  string
	EnableCache bool
}

func BackupOptionsForBackupConfig(backupConfig api.BackupConfiguration, extraOpt ExtraOptions) restic.BackupOptions {
	backupOpt := restic.BackupOptions{
		Host:            extraOpt.Host,
		RetentionPolicy: backupConfig.Spec.RetentionPolicy,
	}
	if backupConfig.Spec.Target != nil {
		backupOpt.BackupDirs = backupConfig.Spec.Target.Directories
	}
	return backupOpt
}

func RestoreOptionForRestoreSession(restoreSession api.RestoreSession, extraOpt ExtraOptions) restic.RestoreOptions {
	return RestoreOptionsForHost(extraOpt.Host, restoreSession.Spec.Rules)
}

// return first matching rule
// if hosts is empty for any rule, it will match any hostname
func RestoreOptionsForHost(hostname string, rules []api.Rule) restic.RestoreOptions {
	for _, rule := range rules {
		// if host is specified in rule then use it. otherwise use workload itself as host
		sourceHost := hostname
		if rule.SourceHost != "" {
			sourceHost = rule.SourceHost
		}

		if len(rule.Subjects) == 0 || go_str.Contains(rule.Subjects, hostname) {
			return restic.RestoreOptions{
				Host:        hostname,
				SourceHost:  sourceHost,
				RestoreDirs: rule.Paths,
				Snapshots:   rule.Snapshots,
			}
		}
	}
	return restic.RestoreOptions{}
}

func SetupOptionsForRepository(repository api_v1alpha1.Repository, extraOpt ExtraOptions) (restic.SetupOptions, error) {
	provider, err := GetProvider(repository.Spec.Backend)
	if err != nil {
		return restic.SetupOptions{}, err
	}
	bucket, prefix, err := GetBucketAndPrefix(&repository.Spec.Backend)
	if err != nil {
		return restic.SetupOptions{}, err
	}
	return restic.SetupOptions{
		Provider:       provider,
		Bucket:         bucket,
		Path:           prefix,
		Endpoint:       GetEndpoint(&repository.Spec.Backend),
		CacertFile:     extraOpt.CacertFile,
		SecretDir:      extraOpt.SecretDir,
		ScratchDir:     extraOpt.ScratchDir,
		EnableCache:    extraOpt.EnableCache,
		MaxConnections: GetMaxConnections(repository.Spec.Backend),
	}, nil
}

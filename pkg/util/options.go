package util

import (
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/restic"

	go_str "github.com/appscode/go/strings"
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
		backupOpt.BackupPaths = backupConfig.Spec.Target.Paths
	}
	return backupOpt
}

func RestoreOptionForRestoreSession(restoreSession api.RestoreSession, extraOpt ExtraOptions) restic.RestoreOptions {
	return RestoreOptionsForHost(extraOpt.Host, restoreSession.Spec.Rules)
}

// return the matching rule
// if targetHosts is empty for a rule, it will match any hostname
func RestoreOptionsForHost(hostname string, rules []api.Rule) restic.RestoreOptions {
	var matchedRule restic.RestoreOptions
	// first check for rules non-empty targetHost
	for _, rule := range rules {
		// if sourceHost is specified in the rule then use it. otherwise use workload itself as host
		sourceHost := hostname
		if rule.SourceHost != "" {
			sourceHost = rule.SourceHost
		}

		if len(rule.TargetHosts) == 0 || go_str.Contains(rule.TargetHosts, hostname) {
			matchedRule = restic.RestoreOptions{
				Host:         hostname,
				SourceHost:   sourceHost,
				RestorePaths: rule.Paths,
				Snapshots:    rule.Snapshots,
			}
			// if rule has empty targetHost then check further rules to see if any other rule with non-empty targetHost matches
			if len(rule.TargetHosts) == 0 {
				continue
			} else {
				return matchedRule
			}
		}
	}
	// matchedRule is either emtpy or contains restore option for the rules with empty targetHost field.
	return matchedRule
}

func SetupOptionsForRepository(repository api_v1alpha1.Repository, extraOpt ExtraOptions) (restic.SetupOptions, error) {
	provider, err := repository.Spec.Backend.Provider()
	if err != nil {
		return restic.SetupOptions{}, err
	}
	bucket, err := repository.Spec.Backend.Container()
	if err != nil {
		return restic.SetupOptions{}, err
	}
	prefix, err := repository.Spec.Backend.Prefix()
	if err != nil {
		return restic.SetupOptions{}, err
	}
	endpoint, _ := repository.Spec.Backend.Endpoint()

	return restic.SetupOptions{
		Provider:       provider,
		Bucket:         bucket,
		Path:           prefix,
		Endpoint:       endpoint,
		CacertFile:     extraOpt.CacertFile,
		SecretDir:      extraOpt.SecretDir,
		ScratchDir:     extraOpt.ScratchDir,
		EnableCache:    extraOpt.EnableCache,
		MaxConnections: repository.Spec.Backend.MaxConnections(),
	}, nil
}

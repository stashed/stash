package restic

import (
	"time"

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (w *ResticWrapper) RunRestore(restoreOptions RestoreOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()

	restoreOutput := &RestoreOutput{
		HostRestoreStats: api_v1beta1.HostRestoreStats{
			Hostname: restoreOptions.Host,
		},
	}

	if len(restoreOptions.Snapshots) != 0 {
		for _, snapshot := range restoreOptions.Snapshots {
			// if snapshot is specified then host and path does not matter.
			if _, err := w.restore("", "", snapshot, restoreOptions.Destination); err != nil {
				return nil, err
			}
		}
	} else if len(restoreOptions.RestoreDirs) != 0 {
		for _, path := range restoreOptions.RestoreDirs {
			if _, err := w.restore(path, restoreOptions.SourceHost, "", restoreOptions.Destination); err != nil {
				return nil, err
			}
		}
	}

	// Restore successful. Read current time and calculate total session duration.
	endTime := time.Now()
	restoreOutput.HostRestoreStats.Duration = endTime.Sub(startTime).String()
	restoreOutput.HostRestoreStats.Phase = api_v1beta1.HostRestoreSucceeded

	return restoreOutput, nil
}

func (w *ResticWrapper) Dump(dumpOptions DumpOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()
	restoreOutput := &RestoreOutput{}

	if _, err := w.dump(dumpOptions); err != nil {
		return nil, err
	}

	// Dump successful. Read current time and calculate total session duration.
	endTime := time.Now()
	restoreOutput.HostRestoreStats.Duration = endTime.Sub(startTime).String()
	restoreOutput.HostRestoreStats.Phase = api_v1beta1.HostRestoreSucceeded

	return restoreOutput, nil
}

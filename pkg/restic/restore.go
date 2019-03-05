package restic

import (
	"time"
)

func (w *ResticWrapper) RunRestore(restoreOptions RestoreOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()
	restoreOutput := &RestoreOutput{}

	if len(restoreOptions.Snapshots) != 0 {
		for _, snapshot := range restoreOptions.Snapshots {
			// if snapshot is specified then host and path does not matter.
			if _, err := w.restore("", "", snapshot); err != nil {
				return nil, err
			}
		}
	} else if len(restoreOptions.RestoreDirs) != 0 {
		for _, path := range restoreOptions.RestoreDirs {
			if _, err := w.restore(path, restoreOptions.Host, ""); err != nil {
				return nil, err
			}
		}
	}

	// Restore successful. Read current time and calculate total session duration.
	endTime := time.Now()
	restoreOutput.SessionDuration = endTime.Sub(startTime).String()

	return restoreOutput, nil
}

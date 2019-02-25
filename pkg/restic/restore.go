package restic

import (
	"time"

	"github.com/appscode/go/strings"
)

func (w *ResticWrapper) RunRestore(restoreOptions RestoreOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()

	restoreOutput := &RestoreOutput{}

	// Restore data according to rules. Restore will proceed only for first matching rule.
	for _, rule := range restoreOptions.Rules {
		// Check if Hosts field is empty. Empty Hosts filed will match any host.
		if len(rule.Hosts) == 0 || strings.Contains(rule.Hosts, w.config.Hostname) {
			if len(rule.Snapshots) != 0 {
				for _, snapshot := range rule.Snapshots {
					// if snapshot is specified then host and path does not matter.
					_, err := w.restore("", "", snapshot)
					if err != nil {
						return nil, err
					}
				}
			} else if len(rule.Paths) != 0 {
				host := ""
				if len(rule.Hosts) != 0 {
					host = w.config.Hostname
				}
				for _, path := range rule.Paths {
					_, err := w.restore(path, host, "")
					if err != nil {
						return nil, err
					}
				}
			}
			// one rule is matched. so, stop rule matching.
			break
		}
	}

	// Restore successful. Read current time and calculate total session duration.
	endTime := time.Now()
	restoreOutput.SessionDuration = endTime.Sub(startTime).String()

	return restoreOutput, nil
}

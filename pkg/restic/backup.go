package restic

import "time"

func (w *ResticWrapper) RunBackup(backupOption BackupOptions) (*BackupOutput, error) {
	// Start clock to measure total session duration
	startTime := time.Now()

	// Initialize restic repository if it does not exist
	_, err := w.initRepositoryIfAbsent()
	if err != nil {
		return nil, err
	}

	backupOutput := &BackupOutput{}

	// Backup all target directories
	for _, dir := range backupOption.BackupDirs {
		out, err := w.backup(dir, backupOption.Host, nil)
		if err != nil {
			return nil, err
		}
		// Extract information from the output of backup command
		err = backupOutput.extractBackupInfo(out, dir)
		if err != nil {
			return nil, err
		}
	}

	// Check repository integrity
	out, err := w.check()
	if err != nil {
		return nil, err
	}
	// Extract information from output of "check" command
	backupOutput.extractCheckInfo(out)

	// Cleanup old snapshot according to retention policy
	out, err = w.cleanup(backupOption.RetentionPolicy)
	if err != nil {
		return nil, err
	}
	// Extract information from output of cleanup command
	err = backupOutput.extractCleanupInfo(out)
	if err != nil {
		return nil, err
	}

	// Read repository statics after cleanup
	out, err = w.stats()
	if err != nil {
		return nil, err
	}
	// Extract information from output of "stats" command
	err = backupOutput.extractStatsInfo(out)
	if err != nil {
		return nil, err
	}

	// Backup complete. Read current time and calculate total session duration.
	endTime := time.Now()
	backupOutput.SessionDuration = endTime.Sub(startTime).String()

	return backupOutput, nil
}

/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package restic

import (
	"sync"
	"time"

	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/util/errors"
)

// RunBackup takes backup, cleanup old snapshots, check repository integrity etc.
// It extract valuable information from respective restic command it runs and return them for further use.
func (w *ResticWrapper) RunBackup(backupOption BackupOptions, targetRef api_v1beta1.TargetRef) (*BackupOutput, error) {
	// Start clock to measure total session duration
	startTime := time.Now()
	backupOutput := &BackupOutput{
		BackupTargetStatus: api_v1beta1.BackupTargetStatus{
			Ref: targetRef,
		},
	}
	// Run backup
	hostStats, err := w.runBackup(backupOption)
	if err != nil {
		return nil, err
	}
	backupOutput.BackupTargetStatus.Stats = []api_v1beta1.HostBackupStats{hostStats}

	for idx, hostStats := range backupOutput.BackupTargetStatus.Stats {
		if hostStats.Hostname == backupOption.Host {
			backupOutput.BackupTargetStatus.Stats[idx].Duration = time.Since(startTime).String()
			backupOutput.BackupTargetStatus.Stats[idx].Phase = api_v1beta1.HostBackupSucceeded
		}
	}
	return backupOutput, nil
}

// RunParallelBackup runs multiple backup in parallel.
// Host must be different for each backup.
func (w *ResticWrapper) RunParallelBackup(backupOptions []BackupOptions, targetRef api_v1beta1.TargetRef, maxConcurrency int) (*BackupOutput, error) {
	// WaitGroup to wait until all go routine finishes
	wg := sync.WaitGroup{}
	// concurrencyLimiter channel is used to limit maximum number simultaneous go routine
	concurrencyLimiter := make(chan bool, maxConcurrency)
	defer close(concurrencyLimiter)

	var (
		backupErrs []error
		mu         sync.Mutex // use lock to avoid racing condition
	)

	backupOutput := &BackupOutput{
		BackupTargetStatus: api_v1beta1.BackupTargetStatus{
			Ref: targetRef,
		},
	}

	for i := range backupOptions {
		// try to send message in concurrencyLimiter channel.
		// if maximum allowed concurrent backup is already running, program control will stuck here.
		concurrencyLimiter <- true

		// starting new go routine. add it to WaitGroup
		wg.Add(1)

		go func(opt BackupOptions, startTime time.Time) {
			// when this go routine completes it task, release a slot from the concurrencyLimiter channel
			// so that another go routine can start. Also, tell the WaitGroup that it is done with its task.
			defer func() {
				<-concurrencyLimiter
				wg.Done()
			}()

			// sh field in ResticWrapper is a pointer. we must not use same w in multiple go routine.
			// otherwise they might enter in a racing condition.
			nw := w.Copy()

			hostStats, err := nw.runBackup(opt)
			if err != nil {
				// acquire lock to make sure no other go routine is writing to backupErr
				mu.Lock()
				backupErrs = append(backupErrs, err)
				mu.Unlock()
				return
			}
			hostStats.Duration = time.Since(startTime).String()
			hostStats.Phase = api_v1beta1.HostBackupSucceeded

			// add hostStats to backupOutput. use lock to avoid racing condition.
			mu.Lock()
			backupOutput.upsertHostBackupStats(hostStats)
			mu.Unlock()
		}(backupOptions[i], time.Now())
	}

	// wait for all the go routines to complete
	wg.Wait()

	if backupErrs != nil {
		return nil, errors.NewAggregate(backupErrs)
	}
	return backupOutput, nil
}

func (w *ResticWrapper) runBackup(backupOption BackupOptions) (api_v1beta1.HostBackupStats, error) {
	hostStats := api_v1beta1.HostBackupStats{
		Hostname: backupOption.Host,
	}

	// fmt.Println("shell: ",w)
	// Backup from stdin
	if len(backupOption.StdinPipeCommands) != 0 {
		out, err := w.backupFromStdin(backupOption)
		if err != nil {
			return hostStats, err
		}
		// Extract information from the output of backup command
		snapStats, err := extractBackupInfo(out, backupOption.StdinFileName)
		if err != nil {
			return hostStats, err
		}
		hostStats.Snapshots = []api_v1beta1.SnapshotStats{snapStats}
		return hostStats, nil
	}

	// Backup all target paths
	for _, path := range backupOption.BackupPaths {
		params := backupParams{
			path:     path,
			host:     backupOption.Host,
			excludes: backupOption.Exclude,
			args:     backupOption.Args,
		}
		out, err := w.backup(params)
		if err != nil {
			return hostStats, err
		}
		// Extract information from the output of backup command
		stats, err := extractBackupInfo(out, path)
		if err != nil {
			return hostStats, err
		}
		hostStats = upsertSnapshotStats(hostStats, stats)
	}

	return hostStats, nil
}

func upsertSnapshotStats(hostStats api_v1beta1.HostBackupStats, snapStats api_v1beta1.SnapshotStats) api_v1beta1.HostBackupStats {
	for i, s := range hostStats.Snapshots {
		// if there is already an entry for this snapshot, then update it
		if s.Name == snapStats.Name {
			hostStats.Snapshots[i] = snapStats
			return hostStats
		}
	}
	// no entry for this snapshot. add a new entry
	hostStats.Snapshots = append(hostStats.Snapshots, snapStats)
	return hostStats
}

func (backupOutput *BackupOutput) upsertHostBackupStats(hostStats api_v1beta1.HostBackupStats) {
	// check if a entry already exist for this host in backupOutput. If exist then update it.
	for i, v := range backupOutput.BackupTargetStatus.Stats {
		if v.Hostname == hostStats.Hostname {
			backupOutput.BackupTargetStatus.Stats[i] = hostStats
			return
		}
	}

	// no entry for this host. add a new entry
	backupOutput.BackupTargetStatus.Stats = append(backupOutput.BackupTargetStatus.Stats, hostStats)
}

func (w *ResticWrapper) RepositoryAlreadyExist() bool {
	return w.repositoryExist()
}

func (w *ResticWrapper) InitializeRepository() error {
	return w.initRepository()
}

func (w *ResticWrapper) ApplyRetentionPolicies(retentionPolicy api_v1alpha1.RetentionPolicy) (*RepositoryStats, error) {
	// Cleanup old snapshots according to retention policy
	out, err := w.cleanup(retentionPolicy, "")
	if err != nil {
		return nil, err
	}
	// Extract information from output of cleanup command
	kept, removed, err := extractCleanupInfo(out)
	if err != nil {
		return nil, err
	}
	return &RepositoryStats{SnapshotCount: kept, SnapshotsRemovedOnLastCleanup: removed}, nil
}

func (w *ResticWrapper) VerifyRepositoryIntegrity() (*RepositoryStats, error) {
	// Check repository integrity
	out, err := w.check()
	if err != nil {
		return nil, err
	}
	// Extract information from output of "check" command
	integrity := extractCheckInfo(out)
	// Read repository statics after cleanup
	out, err = w.stats("")
	if err != nil {
		return nil, err
	}
	// Extract information from output of "stats" command
	repoSize, err := extractStatsInfo(out)
	if err != nil {
		return nil, err
	}
	return &RepositoryStats{Integrity: pointer.BoolP(integrity), Size: repoSize}, nil
}

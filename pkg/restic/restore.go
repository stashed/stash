package restic

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/errors"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

// RunRestore run restore process for a single host.
func (w *ResticWrapper) RunRestore(restoreOptions RestoreOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()

	restoreStats := api_v1beta1.HostRestoreStats{
		Hostname: restoreOptions.Host,
	}

	err := w.runRestore(restoreOptions)
	if err != nil {
		return nil, err
	}

	// Restore successful. Now, calculate total session duration.
	restoreStats.Duration = time.Since(startTime).String()
	restoreStats.Phase = api_v1beta1.HostRestoreSucceeded

	restoreOutput := &RestoreOutput{
		HostRestoreStats: []api_v1beta1.HostRestoreStats{restoreStats},
	}

	return restoreOutput, nil
}

// RunParallelRestore run restore process for multiple hosts in parallel using go routine.
// You can control maximum number of parallel backup using maxConcurrency parameter.
func (w *ResticWrapper) RunParallelRestore(restoreOptions []RestoreOptions, maxConcurrency int) (*RestoreOutput, error) {

	// WaitGroup to wait until all go routine finish
	wg := sync.WaitGroup{}
	// concurrencyLimiter channel is used to limit maximum number simultaneous go routine
	concurrencyLimiter := make(chan bool, maxConcurrency)
	defer close(concurrencyLimiter)

	var (
		restoreErrs []error
		mu          sync.Mutex
	)
	restoreOutput := &RestoreOutput{}

	for i := range restoreOptions {
		// try to send message in concurrencyLimiter channel.
		// if maximum allowed concurrent restore is already running, program control will stuck here.
		concurrencyLimiter <- true

		// starting new go routine. add it to WaitGroup
		wg.Add(1)

		go func(opt RestoreOptions, startTime time.Time) {
			// when this go routine completes it task, release a slot from the concurrencyLimiter channel
			// so that another go routine can start. Also, tell the WaitGroup that it is done with its task.
			defer func() {
				<-concurrencyLimiter
				wg.Done()
			}()

			// sh field in ResticWrapper is a pointer. we must not use same w in multiple go routine.
			// otherwise they might enter in a racing condition.
			nw := w.Copy()

			// run restore
			err := nw.runRestore(opt)
			if err != nil {
				mu.Lock()
				restoreErrs = append(restoreErrs, err)
				mu.Unlock()
				return
			}
			hostStats := api_v1beta1.HostRestoreStats{
				Hostname: opt.Host,
			}
			hostStats.Duration = time.Since(startTime).String()
			hostStats.Phase = api_v1beta1.HostRestoreSucceeded

			// add hostStats to restoreOutput
			mu.Lock()
			restoreOutput.upsertHostRestoreStats(hostStats)
			mu.Unlock()
		}(restoreOptions[i], time.Now())
	}

	// wait for all the go routines to complete
	wg.Wait()

	return restoreOutput, errors.NewAggregate(restoreErrs)
}

// Dump run restore process for a single host and output the restored files in stdout.
func (w *ResticWrapper) Dump(dumpOptions DumpOptions) (*RestoreOutput, error) {
	// Start clock to measure total restore duration
	startTime := time.Now()

	restoreStats := api_v1beta1.HostRestoreStats{}
	if _, err := w.dump(dumpOptions); err != nil {
		return nil, err
	}

	// Dump successful. Now, calculate total session duration.
	restoreStats.Duration = time.Since(startTime).String()
	restoreStats.Phase = api_v1beta1.HostRestoreSucceeded

	restoreOutput := &RestoreOutput{
		HostRestoreStats: []api_v1beta1.HostRestoreStats{restoreStats},
	}
	return restoreOutput, nil
}

// ParallelDump run dump for multiple hosts concurrently using go routine.
// You can control maximum number of parallel restore process using maxConcurrency parameter.
func (w *ResticWrapper) ParallelDump(dumpOptions []DumpOptions, maxConcurrency int) (*RestoreOutput, error) {

	// WaitGroup to wait until all go routine finish
	wg := sync.WaitGroup{}
	// concurrencyLimiter channel is used to limit maximum number simultaneous go routine
	concurrencyLimiter := make(chan bool, maxConcurrency)
	defer close(concurrencyLimiter)

	var (
		restoreErrs []error
		mu          sync.Mutex
	)

	restoreOutput := &RestoreOutput{}

	for i := range dumpOptions {
		// try to send message in concurrencyLimiter channel.
		// if maximum allowed concurrent backup is already running, program control will stuck here.
		concurrencyLimiter <- true

		// starting new go routine. add it to WaitGroup
		wg.Add(1)

		go func(opt DumpOptions, startTime time.Time) {
			// when this go routine completes it task, release a slot from the concurrencyLimiter channel
			// so that another go routine can start. Also, tell the WaitGroup that it is done with its task.
			defer func() {
				<-concurrencyLimiter
				wg.Done()
			}()

			// sh field in ResticWrapper is a pointer. we must not use same w in multiple go routine.
			// otherwise they might enter in a racing condition.
			nw := w.Copy()

			// run restore
			_, err := nw.dump(opt)
			if err != nil {
				mu.Lock()
				restoreErrs = append(restoreErrs, err)
				mu.Unlock()
				return
			}
			hostStats := api_v1beta1.HostRestoreStats{
				Hostname: opt.Host,
			}
			hostStats.Duration = time.Since(startTime).String()
			hostStats.Phase = api_v1beta1.HostRestoreSucceeded

			// add hostStats to restoreOutput
			mu.Lock()
			restoreOutput.upsertHostRestoreStats(hostStats)
			mu.Unlock()
		}(dumpOptions[i], time.Now())
	}

	// wait for all the go routines to complete
	wg.Wait()

	return restoreOutput, errors.NewAggregate(restoreErrs)
}

func (w *ResticWrapper) runRestore(restoreOptions RestoreOptions) error {
	if len(restoreOptions.Snapshots) != 0 {
		for _, snapshot := range restoreOptions.Snapshots {
			// if snapshot is specified then host and path does not matter.
			if _, err := w.restore("", "", snapshot, restoreOptions.Destination); err != nil {
				return err
			}
		}
	} else if len(restoreOptions.RestoreDirs) != 0 {
		for _, path := range restoreOptions.RestoreDirs {
			if _, err := w.restore(path, restoreOptions.SourceHost, "", restoreOptions.Destination); err != nil {
				return err
			}
		}
	}
	return nil
}

func (restoreOutput *RestoreOutput) upsertHostRestoreStats(hostStats api_v1beta1.HostRestoreStats) {

	// check if a entry already exist for this host in restoreOutput. If exist then update it.
	for i, v := range restoreOutput.HostRestoreStats {
		if v.Hostname == hostStats.Hostname {
			restoreOutput.HostRestoreStats[i] = hostStats
			return
		}
	}

	// no entry for this host. add a new entry
	restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, hostStats)
	return
}

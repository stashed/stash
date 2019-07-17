package restic

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

// BackupMetrics defines prometheus metrics for backup setup and individual host backup
type BackupMetrics struct {
	// BackupSetupMetrics indicates whether backup was successfully setup for the target
	BackupSetupMetrics prometheus.Gauge
	// HostBackupMetrics shows metrics related to last backup session of a host
	HostBackupMetrics *HostBackupMetrics
}

// RestoreMetrics defines metrics for restore process for individual hosts
type RestoreMetrics struct {
	// RestoreSuccess show whether the current restore session succeeded or not
	RestoreSuccess prometheus.Gauge
	// SessionDuration show total time taken to complete the restore session
	SessionDuration prometheus.Gauge
}

// HostBackupMetrics defines Prometheus metrics for backup individual hosts
type HostBackupMetrics struct {
	// BackupSuccess show whether the current backup for a host succeeded or not
	BackupSuccess prometheus.Gauge
	// SessionDuration show total time taken to complete the backup session
	SessionDuration prometheus.Gauge
	// DataSize shows total size of the target data to backup (in bytes)
	DataSize prometheus.Gauge
	// DataUploaded shows the amount of data uploaded to the repository in this session (in bytes)
	DataUploaded prometheus.Gauge
	// DataProcessingTime shows total time taken to backup the target data
	DataProcessingTime prometheus.Gauge
	// FileMetrics shows information of backup files
	FileMetrics *FileMetrics
}

// FileMetrics defines Prometheus metrics for target files of a backup process
type FileMetrics struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles prometheus.Gauge
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles prometheus.Gauge
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles prometheus.Gauge
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles prometheus.Gauge
}

// RepositoryMetrics defines Prometheus metrics for Repository state after each backup
type RepositoryMetrics struct {
	// RepoIntegrity shows result of repository integrity check after last backup
	RepoIntegrity prometheus.Gauge
	// RepoSize show size of repository after last backup
	RepoSize prometheus.Gauge
	// SnapshotCount shows number of snapshots stored in the repository
	SnapshotCount prometheus.Gauge
	// SnapshotsRemovedOnLastCleanup shows number of old snapshots cleaned up according to retention policy on last backup session
	SnapshotsRemovedOnLastCleanup prometheus.Gauge
}

func newBackupMetrics(labels prometheus.Labels) *BackupMetrics {

	return &BackupMetrics{
		HostBackupMetrics: &HostBackupMetrics{
			BackupSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "session_success",
					Help:        "Indicates whether the current backup session succeeded or not",
					ConstLabels: labels,
				},
			),
			SessionDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "session_duration_total_seconds",
					Help:        "Total time taken to complete the backup session",
					ConstLabels: labels,
				},
			),
			DataSize: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "data_size_bytes",
					Help:        "Total size of the target data to backup (in bytes)",
					ConstLabels: labels,
				},
			),
			DataUploaded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "data_uploaded_bytes",
					Help:        "Amount of data uploaded to the repository in this session (in bytes)",
					ConstLabels: labels,
				},
			),
			DataProcessingTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "data_processing_time_seconds",
					Help:        "Total time taken to backup the target data",
					ConstLabels: labels,
				},
			),
			FileMetrics: &FileMetrics{
				TotalFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "files_total",
						Help:        "Total number of files that has been backed up",
						ConstLabels: labels,
					},
				),
				NewFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "files_new",
						Help:        "Total number of new files that has been created since last backup",
						ConstLabels: labels,
					},
				),
				ModifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "files_modified",
						Help:        "Total number of files that has been modified since last backup",
						ConstLabels: labels,
					},
				),
				UnmodifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "files_unmodified",
						Help:        "Total number of files that has not been changed since last backup",
						ConstLabels: labels,
					},
				),
			},
		},
	}
}

func newBackupSetupMetrics(labels prometheus.Labels) *BackupMetrics {

	return &BackupMetrics{
		BackupSetupMetrics: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "backup",
				Name:        "setup_success",
				Help:        "Indicates whether backup was successfully setup for the target",
				ConstLabels: labels,
			},
		),
	}
}

func newRepositoryMetrics(labels prometheus.Labels) *RepositoryMetrics {

	return &RepositoryMetrics{
		RepoIntegrity: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "repository",
				Name:        "integrity",
				Help:        "Result of repository integrity check after last backup",
				ConstLabels: labels,
			},
		),
		RepoSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "repository",
				Name:        "size_bytes",
				Help:        "Indicates size of repository after last backup (in bytes)",
				ConstLabels: labels,
			},
		),
		SnapshotCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "repository",
				Name:        "snapshot_count",
				Help:        "Indicates number of snapshots stored in the repository",
				ConstLabels: labels,
			},
		),
		SnapshotsRemovedOnLastCleanup: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "repository",
				Name:        "snapshot_cleaned",
				Help:        "Indicates number of old snapshots cleaned up according to retention policy on last backup session",
				ConstLabels: labels,
			},
		),
	}
}

func newRestoreMetrics(labels prometheus.Labels) *RestoreMetrics {

	return &RestoreMetrics{
		RestoreSuccess: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "restore",
				Name:        "session_success",
				Help:        "Indicates whether the current restore session succeeded or not",
				ConstLabels: labels,
			},
		),
		SessionDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "stash",
				Subsystem:   "restore",
				Name:        "session_duration_total_seconds",
				Help:        "Total time taken to complete the restore session",
				ConstLabels: labels,
			},
		),
	}
}

// HandleBackupSetupMetrics generate and send Prometheus metrics for backup setup
func HandleBackupSetupMetrics(metricOpt MetricsOptions, setupErr error) error {
	labels := metricLabels(metricOpt.Labels)
	metrics := newBackupSetupMetrics(labels)

	if setupErr == nil {
		metrics.BackupSetupMetrics.Set(1)
	} else {
		metrics.BackupSetupMetrics.Set(0)
	}

	// create metric registry
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		metrics.BackupSetupMetrics,
	)
	// send metrics
	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

// HandleMetrics generate and send Prometheus metrics for backup process
func (backupOutput *BackupOutput) HandleMetrics(metricOpt *MetricsOptions, backupErr error) error {
	if backupOutput == nil {
		return fmt.Errorf("invalid backup output")
	}

	// create metric registry
	registry := prometheus.NewRegistry()

	// create metrics for individual hosts
	for _, hostStats := range backupOutput.HostBackupStats {
		// add host name as label
		metricOpt.Labels = append(metricOpt.Labels, fmt.Sprintf("Host=%s", hostStats.Hostname))
		labels := metricLabels(metricOpt.Labels)
		metrics := newBackupMetrics(labels)

		if backupErr == nil && hostStats.Error == "" {
			// set metrics values from backupOutput
			err := metrics.setValues(hostStats)
			if err != nil {
				return err
			}
			metrics.HostBackupMetrics.BackupSuccess.Set(1)
		} else {
			metrics.HostBackupMetrics.BackupSuccess.Set(0)
		}
		registry.MustRegister(
			// register backup session metrics
			metrics.HostBackupMetrics.BackupSuccess,
			metrics.HostBackupMetrics.SessionDuration,
			metrics.HostBackupMetrics.FileMetrics.TotalFiles,
			metrics.HostBackupMetrics.FileMetrics.NewFiles,
			metrics.HostBackupMetrics.FileMetrics.ModifiedFiles,
			metrics.HostBackupMetrics.FileMetrics.UnmodifiedFiles,
			metrics.HostBackupMetrics.DataSize,
			metrics.HostBackupMetrics.DataUploaded,
			metrics.HostBackupMetrics.DataProcessingTime,
		)
	}

	// crete repository metrics
	repoMetrics := newRepositoryMetrics(metricLabels(metricOpt.Labels))
	err := repoMetrics.setValues(backupOutput.RepositoryStats)
	if err != nil {
		return err
	}

	// register repository metrics
	registry.MustRegister(
		repoMetrics.RepoIntegrity,
		repoMetrics.RepoSize,
		repoMetrics.SnapshotCount,
		repoMetrics.SnapshotsRemovedOnLastCleanup,
	)

	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func (restoreOutput *RestoreOutput) HandleMetrics(metricOpt *MetricsOptions, restoreErr error) error {
	if restoreOutput == nil {
		return fmt.Errorf("invalid restore output")
	}

	// create metric registry
	registry := prometheus.NewRegistry()

	// create metrics for each host
	for _, hostStats := range restoreOutput.HostRestoreStats {
		// add host name as label
		metricOpt.Labels = append(metricOpt.Labels, fmt.Sprintf("Host=%s", hostStats.Hostname))
		labels := metricLabels(metricOpt.Labels)
		metrics := newRestoreMetrics(labels)

		if restoreErr == nil && hostStats.Error == "" {
			duration, err := time.ParseDuration(hostStats.Duration)
			if err != nil {
				return err
			}
			metrics.SessionDuration.Set(duration.Seconds())
			metrics.RestoreSuccess.Set(1)
		} else {
			metrics.RestoreSuccess.Set(0)
		}
		registry.MustRegister(
			metrics.SessionDuration,
			metrics.RestoreSuccess,
		)
	}

	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func (backupMetrics *BackupMetrics) setValues(hostOutput api_v1beta1.HostBackupStats) error {
	var (
		totalDataSize        float64
		totalUploadSize      float64
		totalProcessingTime  uint64
		totalFiles           int
		totalNewFiles        int
		totalModifiedFiles   int
		totalUnmodifiedFiles int
	)

	for _, v := range hostOutput.Snapshots {
		dataSizeBytes, err := convertSizeToBytes(v.Size)
		if err != nil {
			return err
		}
		totalDataSize = totalDataSize + dataSizeBytes

		uploadSizeBytes, err := convertSizeToBytes(v.Uploaded)
		if err != nil {
			return err
		}
		totalUploadSize = totalUploadSize + uploadSizeBytes

		processingTimeSeconds, err := convertTimeToSeconds(v.ProcessingTime)
		if err != nil {
			return err
		}
		totalProcessingTime = totalProcessingTime + processingTimeSeconds

		totalFiles = totalFiles + *v.FileStats.TotalFiles
		totalNewFiles = totalNewFiles + *v.FileStats.NewFiles
		totalModifiedFiles = totalModifiedFiles + *v.FileStats.ModifiedFiles
		totalUnmodifiedFiles = totalUnmodifiedFiles + *v.FileStats.UnmodifiedFiles
	}

	backupMetrics.HostBackupMetrics.DataSize.Set(totalDataSize)
	backupMetrics.HostBackupMetrics.DataUploaded.Set(totalUploadSize)
	backupMetrics.HostBackupMetrics.DataProcessingTime.Set(float64(totalProcessingTime))
	backupMetrics.HostBackupMetrics.FileMetrics.TotalFiles.Set(float64(totalFiles))
	backupMetrics.HostBackupMetrics.FileMetrics.NewFiles.Set(float64(totalNewFiles))
	backupMetrics.HostBackupMetrics.FileMetrics.ModifiedFiles.Set(float64(totalModifiedFiles))
	backupMetrics.HostBackupMetrics.FileMetrics.UnmodifiedFiles.Set(float64(totalUnmodifiedFiles))

	duration, err := time.ParseDuration(hostOutput.Duration)
	if err != nil {
		return err
	}
	backupMetrics.HostBackupMetrics.SessionDuration.Set(duration.Seconds())

	return nil
}

func (repoMetrics *RepositoryMetrics) setValues(repoStats RepositoryStats) error {
	// set repository metrics values
	if *repoStats.Integrity {
		repoMetrics.RepoIntegrity.Set(1)
	} else {
		repoMetrics.RepoIntegrity.Set(0)
	}
	repoSize, err := convertSizeToBytes(repoStats.Size)
	if err != nil {
		return err
	}
	repoMetrics.RepoSize.Set(repoSize)
	repoMetrics.SnapshotCount.Set(float64(repoStats.SnapshotCount))
	repoMetrics.SnapshotsRemovedOnLastCleanup.Set(float64(repoStats.SnapshotsRemovedOnLastCleanup))

	return nil
}

func (metricOpt *MetricsOptions) sendMetrics(registry *prometheus.Registry, jobName string) error {
	// if Pushgateway URL is provided, then push the metrics to Pushgateway
	if metricOpt.PushgatewayURL != "" {
		pusher := push.New(metricOpt.PushgatewayURL, jobName)
		err := pusher.Gatherer(registry).Push()
		if err != nil {
			return err
		}
	}

	// if metric file directory is specified, then write the metrics in "metric.prom" text file in the specified directory
	if metricOpt.MetricFileDir != "" {
		err := prometheus.WriteToTextfile(filepath.Join(metricOpt.MetricFileDir, "metric.prom"), registry)
		if err != nil {
			return err
		}
	}
	return nil
}

func metricLabels(labels []string) prometheus.Labels {
	promLabels := prometheus.Labels{}
	for _, v := range labels {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			promLabels[parts[0]] = parts[1]
		}
	}
	return promLabels
}

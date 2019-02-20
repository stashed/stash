package restic

import (
	"path/filepath"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Metrics struct {
	// BackupMetrics shows metrics related to last backup session
	BackupMetrics *BackupMetrics
	// RepositoryMetrics shows metrics related to repository after last backup
	RepositoryMetrics *RepositoryMetrics
}

type BackupMetrics struct {
	// BackupSuccess show weather the current backup session succeeded or not
	BackupSuccess prometheus.Gauge
	// DataSize shows total size of the target data to backup (in bytes)
	DataSize prometheus.Gauge
	// DataUploaded shows the amount of data uploaded to the repository in this session (in bytes)
	DataUploaded prometheus.Gauge
	// DataProcessingTime shows total time taken to backup the target data
	DataProcessingTime prometheus.Gauge
	// FileMetrics shows information of backup files
	FileMetrics *FileMetrics
}
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

type RepositoryMetrics struct {
	// RepoIntegrity shows result of repository integrity check after last backup
	RepoIntegrity prometheus.Gauge
	// RepoSize show size of repository after last backup
	RepoSize prometheus.Gauge
	// SnapshotCount shows number of snapshots stored in the repository
	SnapshotCount prometheus.Gauge
	// SnapshotRemovedOnLastCleanup shows number of old snapshots cleaned up according to retention policy on last backup session
	SnapshotRemovedOnLastCleanup prometheus.Gauge
}
type MetricsOptions struct {
	Enabled        bool
	PushgatewayURL string
	MetricFileDir  string
	Labels         []string
}

func NewMetrics(labels prometheus.Labels) *Metrics {

	return &Metrics{
		BackupMetrics: &BackupMetrics{
			BackupSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "success",
					Help:        "Indicates weather the current backup session succeeded or not",
					ConstLabels: labels,
				},
			),
			DataSize: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "data_size_bytes",
					Help:        "Total size of the target data to backup (in bytes)",
					ConstLabels: labels,
				},
			),
			DataUploaded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "data_uploaded_bytes",
					Help:        "Amount of data uploaded to the repository in this session (in bytes)",
					ConstLabels: labels,
				},
			),
			DataProcessingTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "data_processing_time_seconds",
					Help:        "Total time taken to backup the target data",
					ConstLabels: labels,
				},
			),
			FileMetrics: &FileMetrics{
				TotalFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "restic",
						Subsystem:   "backup",
						Name:        "total_files",
						Help:        "Total number of files that has been backed up",
						ConstLabels: labels,
					},
				),
				NewFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "restic",
						Subsystem:   "backup",
						Name:        "new_files",
						Help:        "Total number of new files that has been created since last backup",
						ConstLabels: labels,
					},
				),
				ModifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "restic",
						Subsystem:   "backup",
						Name:        "modified_files",
						Help:        "Total number of files that has been modified since last backup",
						ConstLabels: labels,
					},
				),
				UnmodifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "restic",
						Subsystem:   "backup",
						Name:        "unmodified_files",
						Help:        "Total number of files that has not been changed since last backup",
						ConstLabels: labels,
					},
				),
			},
		},
		RepositoryMetrics: &RepositoryMetrics{
			RepoIntegrity: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "repository",
					Name:        "integrity",
					Help:        "Result of repository integrity check after last backup",
					ConstLabels: labels,
				},
			),
			RepoSize: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "repository",
					Name:        "size_bytes",
					Help:        "Indicates size of repository after last backup (in bytes)",
					ConstLabels: labels,
				},
			),
			SnapshotCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "repository",
					Name:        "snapshot_count",
					Help:        "Indicates number of snapshots stored in the repository",
					ConstLabels: labels,
				},
			),
			SnapshotRemovedOnLastCleanup: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "repository",
					Name:        "snapshot_removed_on_last_cleanup",
					Help:        "Indicates number of old snapshots cleaned up according to retention policy on last backup session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func (metrics *Metrics) SetValues(backupOutput *BackupOutput) error {
	// set backup metrics values
	dataSizeBytes, err := convertSizeToBytes(backupOutput.BackupStats.Size)
	if err != nil {
		return err
	}
	metrics.BackupMetrics.DataSize.Set(dataSizeBytes)

	uploadSizeBytes, err := convertSizeToBytes(backupOutput.BackupStats.Uploaded)
	if err != nil {
		return err
	}
	metrics.BackupMetrics.DataUploaded.Set(uploadSizeBytes)

	processingTimeSeconds, err := convertTimeToSeconds(backupOutput.BackupStats.ProcessingTime)
	if err != nil {
		return err
	}
	metrics.BackupMetrics.DataProcessingTime.Set(float64(processingTimeSeconds))

	metrics.BackupMetrics.FileMetrics.TotalFiles.Set(float64(*backupOutput.BackupStats.FileStats.TotalFiles))
	metrics.BackupMetrics.FileMetrics.NewFiles.Set(float64(*backupOutput.BackupStats.FileStats.NewFiles))
	metrics.BackupMetrics.FileMetrics.ModifiedFiles.Set(float64(*backupOutput.BackupStats.FileStats.ModifiedFiles))
	metrics.BackupMetrics.FileMetrics.UnmodifiedFiles.Set(float64(*backupOutput.BackupStats.FileStats.UnmodifiedFiles))

	// set repository metrics values
	if *backupOutput.RepositoryStats.Integrity {
		metrics.RepositoryMetrics.RepoIntegrity.Set(1)
	} else {
		metrics.RepositoryMetrics.RepoIntegrity.Set(0)
	}
	repoSize, err := convertSizeToBytes(backupOutput.RepositoryStats.Size)
	if err != nil {
		return err
	}
	metrics.RepositoryMetrics.RepoSize.Set(repoSize)
	metrics.RepositoryMetrics.SnapshotCount.Set(float64(backupOutput.RepositoryStats.SnapshotCount))
	metrics.RepositoryMetrics.SnapshotRemovedOnLastCleanup.Set(float64(backupOutput.RepositoryStats.SnapshotRemovedOnLastCleanup))
	return nil
}

func (metricOpt *MetricsOptions) HandleMetrics(backupOutput *BackupOutput, backupErr error, jobName string) error {
	labels := prometheus.Labels{}
	for _, v := range metricOpt.Labels {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	metrics := NewMetrics(labels)

	if backupErr == nil {
		// set metrics values from backupOutput
		err := metrics.SetValues(backupOutput)
		if err != nil {
			return err
		}
		metrics.BackupMetrics.BackupSuccess.Set(1)
	} else {
		metrics.BackupMetrics.BackupSuccess.Set(0)
	}

	// crate metric registry
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		// register backup metrics
		metrics.BackupMetrics.FileMetrics.TotalFiles,
		metrics.BackupMetrics.FileMetrics.NewFiles,
		metrics.BackupMetrics.FileMetrics.ModifiedFiles,
		metrics.BackupMetrics.FileMetrics.UnmodifiedFiles,
		metrics.BackupMetrics.DataSize,
		metrics.BackupMetrics.DataUploaded,
		metrics.BackupMetrics.DataProcessingTime,
		metrics.BackupMetrics.BackupSuccess,
		// register repository metrics
		metrics.RepositoryMetrics.RepoIntegrity,
		metrics.RepositoryMetrics.RepoSize,
		metrics.RepositoryMetrics.SnapshotCount,
		metrics.RepositoryMetrics.SnapshotRemovedOnLastCleanup,
	)

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

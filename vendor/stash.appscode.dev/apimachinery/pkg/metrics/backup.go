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

package metrics

import (
	"fmt"
	"time"

	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"
)

// BackupMetrics defines prometheus metrics for backup process
type BackupMetrics struct {
	// BackupSessionMetrics shows metrics related to entire backup session
	BackupSessionMetrics *BackupSessionMetrics
	// BackupTargetMetrics shows metrics related to a target
	BackupTargetMetrics *BackupTargetMetrics
	// BackupHostMetrics shows backup metrics for individual hosts
	BackupHostMetrics *BackupHostMetrics
}

// BackupSessionMetrics defines metrics for entire backup session
type BackupSessionMetrics struct {
	// SessionSuccess indicates whether the entire backup session was succeeded or not
	SessionSuccess prometheus.Gauge
	// SessionDuration indicates total time taken to complete the entire backup session
	SessionDuration prometheus.Gauge
	// TargetCount indicates the total number of targets that was backed up in this backup session
	TargetCount prometheus.Gauge
	// LastSuccessTime indicates the time(in unix epoch) when the last BackupSession was succeeded
	LastSuccessTime prometheus.Gauge
}

// BackupTargetMetrics defines metrics related to a target
type BackupTargetMetrics struct {
	// TargetBackupSucceeded indicates whether the backup for a target has succeeded or not
	TargetBackupSucceeded prometheus.Gauge
	// HostCount indicates the total number of hosts that was backed up for this target
	HostCount prometheus.Gauge
	// LastSuccessTime indicates the time (in unix epoch) when the last backup was successful for this target
	LastSuccessTime prometheus.Gauge
}

// BackupHostMetrics defines Prometheus metrics for individual hosts backup
type BackupHostMetrics struct {
	// BackupSuccess indicates whether the backup for a host succeeded or not
	BackupSuccess prometheus.Gauge
	// BackupDuration indicates total time taken to complete the backup process for a host
	BackupDuration prometheus.Gauge
	// DataSize indicates total size of the target data to backup for a host (in bytes)
	DataSize prometheus.Gauge
	// DataUploaded indicates the amount of data uploaded to the repository for a host (in bytes)
	DataUploaded prometheus.Gauge
	// DataProcessingTime indicates total time taken to backup the target data for a host
	DataProcessingTime prometheus.Gauge
	// FileMetrics shows information of backup files
	FileMetrics *FileMetrics
}

// FileMetrics defines Prometheus metrics for target files of a backup process for a host
type FileMetrics struct {
	// TotalFiles shows total number of files that has been backed up for a host
	TotalFiles prometheus.Gauge
	// NewFiles shows total number of new files that has been created since last backup for a host
	NewFiles prometheus.Gauge
	// ModifiedFiles shows total number of files that has been modified since last backup for a host
	ModifiedFiles prometheus.Gauge
	// UnmodifiedFiles shows total number of files that has not been changed since last backup for a host
	UnmodifiedFiles prometheus.Gauge
}

func legacyBackupSessionMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupSessionMetrics: &BackupSessionMetrics{
			SessionSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "session_success",
					Help:        "Indicates whether the entire backup session was succeeded or not",
					ConstLabels: labels,
				},
			),
			SessionDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "session_duration_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
			TargetCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "target_count_total",
					Help:        "Indicates the total number of target that was backed up in this backup session",
					ConstLabels: labels,
				},
			),
			LastSuccessTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "last_success_time_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func newBackupSessionMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupSessionMetrics: &BackupSessionMetrics{
			SessionSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "success",
					Help:        "Indicates whether the entire backup session was succeeded or not",
					ConstLabels: labels,
				},
			),
			SessionDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "duration_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
			TargetCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "target_count_total",
					Help:        "Indicates the total number of target that was backed up in this backup session",
					ConstLabels: labels,
				},
			),
			LastSuccessTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "last_success_time_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func legacyBackupTargetMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupTargetMetrics: &BackupTargetMetrics{
			TargetBackupSucceeded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "target_success",
					Help:        "Indicates whether the backup for a target has succeeded or not",
					ConstLabels: labels,
				},
			),
			HostCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "target_host_count_total",
					Help:        "Indicates the total number of hosts that was backed up for this target",
					ConstLabels: labels,
				},
			),
			LastSuccessTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "target_last_success_time_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func newBackupTargetMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupTargetMetrics: &BackupTargetMetrics{
			TargetBackupSucceeded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "target_success",
					Help:        "Indicates whether the backup for a target has succeeded or not",
					ConstLabels: labels,
				},
			),
			HostCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "target_host_count_total",
					Help:        "Indicates the total number of hosts that was backed up for this target",
					ConstLabels: labels,
				},
			),
			LastSuccessTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "target_last_success_time_seconds",
					Help:        "Indicates total time taken to complete the entire backup session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func legacyBackupHostMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupHostMetrics: &BackupHostMetrics{
			BackupSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "host_backup_success",
					Help:        "Indicates whether the backup for a host succeeded or not",
					ConstLabels: labels,
				},
			),
			BackupDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "host_backup_duration_seconds",
					Help:        "Indicates total time taken to complete the backup process for a host",
					ConstLabels: labels,
				},
			),
			DataSize: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "host_data_size_bytes",
					Help:        "Total size of the target data to backup for a host (in bytes)",
					ConstLabels: labels,
				},
			),
			DataUploaded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "host_data_uploaded_bytes",
					Help:        "Amount of data uploaded to the repository for a host (in bytes)",
					ConstLabels: labels,
				},
			),
			DataProcessingTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "backup",
					Name:        "host_data_processing_time_seconds",
					Help:        "Total time taken to process the target data for a host",
					ConstLabels: labels,
				},
			),
			FileMetrics: &FileMetrics{
				TotalFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "host_files_total",
						Help:        "Total number of files that has been backed up for a host",
						ConstLabels: labels,
					},
				),
				NewFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "host_files_new",
						Help:        "Total number of new files that has been created since last backup for a host",
						ConstLabels: labels,
					},
				),
				ModifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "host_files_modified",
						Help:        "Total number of files that has been modified since last backup for a host",
						ConstLabels: labels,
					},
				),
				UnmodifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash",
						Subsystem:   "backup",
						Name:        "host_files_unmodified",
						Help:        "Total number of files that has not been changed since last backup for a host",
						ConstLabels: labels,
					},
				),
			},
		},
	}
}

func newBackupHostMetrics(labels prometheus.Labels) *BackupMetrics {
	return &BackupMetrics{
		BackupHostMetrics: &BackupHostMetrics{
			BackupSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "host_backup_success",
					Help:        "Indicates whether the backup for a host succeeded or not",
					ConstLabels: labels,
				},
			),
			BackupDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "host_backup_duration_seconds",
					Help:        "Indicates total time taken to complete the backup process for a host",
					ConstLabels: labels,
				},
			),
			DataSize: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "host_data_size_bytes",
					Help:        "Total size of the target data to backup for a host (in bytes)",
					ConstLabels: labels,
				},
			),
			DataUploaded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "host_data_uploaded_bytes",
					Help:        "Amount of data uploaded to the repository for a host (in bytes)",
					ConstLabels: labels,
				},
			),
			DataProcessingTime: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "backupsession",
					Name:        "host_data_processing_time_seconds",
					Help:        "Total time taken to process the target data for a host",
					ConstLabels: labels,
				},
			),
			FileMetrics: &FileMetrics{
				TotalFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash_appscode_com",
						Subsystem:   "backupsession",
						Name:        "host_files_total",
						Help:        "Total number of files that has been backed up for a host",
						ConstLabels: labels,
					},
				),
				NewFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash_appscode_com",
						Subsystem:   "backupsession",
						Name:        "host_files_new",
						Help:        "Total number of new files that has been created since last backup for a host",
						ConstLabels: labels,
					},
				),
				ModifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash_appscode_com",
						Subsystem:   "backupsession",
						Name:        "host_files_modified",
						Help:        "Total number of files that has been modified since last backup for a host",
						ConstLabels: labels,
					},
				),
				UnmodifiedFiles: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Namespace:   "stash_appscode_com",
						Subsystem:   "backupsession",
						Name:        "host_files_unmodified",
						Help:        "Total number of files that has not been changed since last backup for a host",
						ConstLabels: labels,
					},
				),
			},
		},
	}
}

// SendBackupSessionMetrics send backup session related metrics to the Pushgateway
func (metricOpt *MetricsOptions) SendBackupSessionMetrics(inv invoker.BackupInvoker, status api_v1beta1.BackupSessionStatus) error {
	// create metric registry
	registry := prometheus.NewRegistry()

	// generate metrics labels
	labels, err := backupInvokerLabels(inv, metricOpt.Labels)
	if err != nil {
		return err
	}

	err = exportBackupSessionMetrics(labels, status, registry)
	if err != nil {
		return err
	}

	err = exportBackupSessionLegacyMetrics(labels, status, registry)
	if err != nil {
		return err
	}

	// send metrics to the pushgateway
	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func exportBackupSessionMetrics(labels prometheus.Labels, status api_v1beta1.BackupSessionStatus, registry *prometheus.Registry) error {
	metrics := newBackupSessionMetrics(labels)
	return setBackupSessionMetrics(metrics, status, registry)
}

func exportBackupSessionLegacyMetrics(labels prometheus.Labels, status api_v1beta1.BackupSessionStatus, registry *prometheus.Registry) error {
	metrics := legacyBackupSessionMetrics(labels)
	return setBackupSessionMetrics(metrics, status, registry)
}

func checkIfBackupSessionSucceeded(status api_v1beta1.BackupSessionStatus) bool {
	for _, tr := range status.Targets {
		if tr.Phase != api_v1beta1.TargetBackupSucceeded {
			return false
		}
	}
	return true
}

func setBackupSessionMetrics(metrics *BackupMetrics, status api_v1beta1.BackupSessionStatus, registry *prometheus.Registry) error {
	if checkIfBackupSessionSucceeded(status) {
		metrics.BackupSessionMetrics.SessionSuccess.Set(1)

		// set total time taken to complete the entire backup session
		duration, err := time.ParseDuration(status.SessionDuration)
		if err != nil {
			return err
		}
		metrics.BackupSessionMetrics.SessionDuration.Set(duration.Seconds())

		// set total number of target that was backed up in this backup session
		metrics.BackupSessionMetrics.TargetCount.Set(float64(len(status.Targets)))

		// set last successful session time to current time
		metrics.BackupSessionMetrics.LastSuccessTime.SetToCurrentTime()

		// register metrics to the registry
		registry.MustRegister(
			metrics.BackupSessionMetrics.SessionSuccess,
			metrics.BackupSessionMetrics.SessionDuration,
			metrics.BackupSessionMetrics.TargetCount,
			metrics.BackupSessionMetrics.LastSuccessTime,
		)
	} else {
		// mark entire backup session as failed
		metrics.BackupSessionMetrics.SessionSuccess.Set(0)
		registry.MustRegister(metrics.BackupSessionMetrics.SessionSuccess)
	}

	return nil
}

// SendBackupTargetMetrics send backup target metrics to the Pushgateway
func (metricOpt *MetricsOptions) SendBackupTargetMetrics(config *rest.Config, i invoker.BackupInvoker, targetRef api_v1beta1.TargetRef, status api_v1beta1.BackupSessionStatus) error {
	// create metric registry
	registry := prometheus.NewRegistry()

	// generate backup session related labels
	labels, err := backupInvokerLabels(i, metricOpt.Labels)
	if err != nil {
		return err
	}
	// generate target related labels
	targetLabels, err := targetLabels(config, targetRef, targetRef.Namespace)
	if err != nil {
		return err
	}
	labels = upsertLabel(labels, targetLabels)

	// only send the metric for the target specified by targetRef
	for _, targetStatus := range status.Targets {
		if invoker.TargetMatched(targetStatus.Ref, targetRef) {
			exportBackupTargetMetrics(labels, targetStatus, registry)
			exportBackupTargetLegacyMetrics(labels, targetStatus, registry)

			// send metrics to the pushgateway
			return metricOpt.sendMetrics(registry, metricOpt.JobName)
		}
	}
	return nil
}

func exportBackupTargetMetrics(labels prometheus.Labels, targetStatus api_v1beta1.BackupTargetStatus, registry *prometheus.Registry) {
	metrics := newBackupTargetMetrics(labels)
	setBackupTargetMetrics(metrics, targetStatus, registry)
}

func exportBackupTargetLegacyMetrics(labels prometheus.Labels, targetStatus api_v1beta1.BackupTargetStatus, registry *prometheus.Registry) {
	metrics := legacyBackupTargetMetrics(labels)
	setBackupTargetMetrics(metrics, targetStatus, registry)
}

func setBackupTargetMetrics(metrics *BackupMetrics, targetStatus api_v1beta1.BackupTargetStatus, registry *prometheus.Registry) {
	if targetStatus.Phase == api_v1beta1.TargetBackupSucceeded {
		// mark target backup as succeeded
		metrics.BackupTargetMetrics.TargetBackupSucceeded.Set(1)

		// set last successful backup time for this target to current time
		metrics.BackupTargetMetrics.LastSuccessTime.SetToCurrentTime()

		// set total number of target that was backed up in this backup session
		if targetStatus.TotalHosts != nil {
			metrics.BackupTargetMetrics.HostCount.Set(float64(*targetStatus.TotalHosts))
		}

		// register metrics to the registry
		registry.MustRegister(
			metrics.BackupTargetMetrics.TargetBackupSucceeded,
			metrics.BackupTargetMetrics.LastSuccessTime,
			metrics.BackupTargetMetrics.HostCount,
		)
	} else {
		// mark target backup as failed
		metrics.BackupTargetMetrics.TargetBackupSucceeded.Set(0)
		registry.MustRegister(metrics.BackupTargetMetrics.TargetBackupSucceeded)
	}
}

// SendBackupHostMetrics send backup metrics for individual hosts to the Pushgateway
func (metricOpt *MetricsOptions) SendBackupHostMetrics(config *rest.Config, i invoker.BackupInvoker, targetRef api_v1beta1.TargetRef, backupOutput *restic.BackupOutput) error {
	if backupOutput == nil {
		return fmt.Errorf("invalid backup output. Backup output shouldn't be nil")
	}

	// create metric registry
	registry := prometheus.NewRegistry()

	// generate backup session related labels
	labels, err := backupInvokerLabels(i, metricOpt.Labels)
	if err != nil {
		return err
	}
	// generate target related labels
	targetLabels, err := targetLabels(config, targetRef, i.GetObjectMeta().Namespace)
	if err != nil {
		return err
	}
	labels = upsertLabel(labels, targetLabels)

	// create metrics for the individual host
	for _, hostStats := range backupOutput.BackupTargetStatus.Stats {
		// add host name as label
		hostLabel := map[string]string{
			MetricLabelHostname: hostStats.Hostname,
		}

		metricLabels := upsertLabel(labels, hostLabel)

		err := exportBackupHostMetrics(metricLabels, hostStats, registry)
		if err != nil {
			return err
		}

		err = exportBackupHostLegacyMetrics(metricLabels, hostStats, registry)
		if err != nil {
			return err
		}
	}
	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func exportBackupHostMetrics(labels prometheus.Labels, hostStats api_v1beta1.HostBackupStats, registry *prometheus.Registry) error {
	metrics := newBackupHostMetrics(labels)
	return setBackupHostMetrics(metrics, hostStats, registry)
}

func exportBackupHostLegacyMetrics(labels prometheus.Labels, hostStats api_v1beta1.HostBackupStats, registry *prometheus.Registry) error {
	metrics := legacyBackupHostMetrics(labels)
	return setBackupHostMetrics(metrics, hostStats, registry)
}

func setBackupHostMetrics(metrics *BackupMetrics, hostStats api_v1beta1.HostBackupStats, registry *prometheus.Registry) error {
	if hostStats.Error == "" {
		// set metrics values from backupOutput
		err := metrics.setValues(hostStats)
		if err != nil {
			return err
		}
		metrics.BackupHostMetrics.BackupSuccess.Set(1)

		registry.MustRegister(
			// register backup session metrics
			metrics.BackupHostMetrics.BackupSuccess,
			metrics.BackupHostMetrics.BackupDuration,
			metrics.BackupHostMetrics.FileMetrics.TotalFiles,
			metrics.BackupHostMetrics.FileMetrics.NewFiles,
			metrics.BackupHostMetrics.FileMetrics.ModifiedFiles,
			metrics.BackupHostMetrics.FileMetrics.UnmodifiedFiles,
			metrics.BackupHostMetrics.DataSize,
			metrics.BackupHostMetrics.DataUploaded,
			metrics.BackupHostMetrics.DataProcessingTime,
		)
	} else {
		metrics.BackupHostMetrics.BackupSuccess.Set(0)

		registry.MustRegister(
			metrics.BackupHostMetrics.BackupSuccess,
		)
	}

	return nil
}

// nolint:unparam
func backupInvokerLabels(inv invoker.BackupInvoker, userProvidedLabels []string) (prometheus.Labels, error) {
	// add user provided labels
	promLabels := parseUserProvidedLabels(userProvidedLabels)

	// add invoker information
	promLabels[MetricLabelInvokerKind] = inv.GetTypeMeta().Kind
	promLabels[MetricLabelInvokerName] = inv.GetObjectMeta().Name
	promLabels[MetricsLabelNamespace] = inv.GetObjectMeta().Namespace

	// insert target information as metrics label
	if inv.GetDriver() == api_v1beta1.VolumeSnapshotter {
		promLabels = upsertLabel(promLabels, volumeSnapshotterLabels())
	} else {
		promLabels[MetricsLabelDriver] = string(api_v1beta1.ResticSnapshotter)
		promLabels[MetricsLabelRepository] = inv.GetRepoRef().Name
	}

	return promLabels, nil
}

func (backupMetrics *BackupMetrics) setValues(hostOutput api_v1beta1.HostBackupStats) error {
	var (
		totalDataSize        float64
		totalUploadSize      float64
		totalProcessingTime  uint64
		totalFiles           int64
		totalNewFiles        int64
		totalModifiedFiles   int64
		totalUnmodifiedFiles int64
	)

	for _, v := range hostOutput.Snapshots {
		dataSizeBytes, err := convertSizeToBytes(v.TotalSize)
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

	backupMetrics.BackupHostMetrics.DataSize.Set(totalDataSize)
	backupMetrics.BackupHostMetrics.DataUploaded.Set(totalUploadSize)
	backupMetrics.BackupHostMetrics.DataProcessingTime.Set(float64(totalProcessingTime))
	backupMetrics.BackupHostMetrics.FileMetrics.TotalFiles.Set(float64(totalFiles))
	backupMetrics.BackupHostMetrics.FileMetrics.NewFiles.Set(float64(totalNewFiles))
	backupMetrics.BackupHostMetrics.FileMetrics.ModifiedFiles.Set(float64(totalModifiedFiles))
	backupMetrics.BackupHostMetrics.FileMetrics.UnmodifiedFiles.Set(float64(totalUnmodifiedFiles))

	duration, err := time.ParseDuration(hostOutput.Duration)
	if err != nil {
		return err
	}
	backupMetrics.BackupHostMetrics.BackupDuration.Set(duration.Seconds())

	return nil
}

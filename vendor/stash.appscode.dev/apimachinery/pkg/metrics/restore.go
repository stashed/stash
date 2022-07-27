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

// RestoreMetrics defines metrics for the restore process
type RestoreMetrics struct {
	// RestoreSessionMetrics shows metrics related to entire restore session
	RestoreSessionMetrics *RestoreSessionMetrics
	// RestoreTargetMetrics shows metrics related to a restore target
	RestoreTargetMetrics *RestoreTargetMetrics
	// RestoreHostMetrics shows metrics related to the individual host of a restore target
	RestoreHostMetrics *RestoreHostMetrics
}

// RestoreSessionMetrics defines metrics related to entire restore session
type RestoreSessionMetrics struct {
	// SessionSuccess indicates whether the restore session succeeded or not
	SessionSuccess prometheus.Gauge
	// SessionDuration indicates the total time taken to complete the entire restore session
	SessionDuration prometheus.Gauge
	// TargetCount indicates the number of targets that was restored in this restore session
	TargetCount prometheus.Gauge
}

// RestoreTargetMetrics defines metrics related to a restore target
type RestoreTargetMetrics struct {
	// TargetRestoreSucceeded indicates whether the restore for a target has succeeded or not
	TargetRestoreSucceeded prometheus.Gauge
	// HostCount indicates the total number of hosts that was restored up for a restore target
	HostCount prometheus.Gauge
}

// RestoreHostMetrics defines restore metrics for the individual hosts
type RestoreHostMetrics struct {
	// RestoreSuccess indicates whether restore was succeeded or not for a host
	RestoreSuccess prometheus.Gauge
	// RestoreDuration indicates the time taken to complete the restore process for a host
	RestoreDuration prometheus.Gauge
}

func newRestoreSessionMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreSessionMetrics: &RestoreSessionMetrics{
			SessionSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "success",
					Help:        "Indicates whether the entire restore session was succeeded or not",
					ConstLabels: labels,
				},
			),
			SessionDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "duration_seconds",
					Help:        "Indicates the total time taken to complete the entire restore session",
					ConstLabels: labels,
				},
			),
			TargetCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "target_count_total",
					Help:        "Indicates the total number of targets that was restored in this restore session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func legacyRestoreSessionMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreSessionMetrics: &RestoreSessionMetrics{
			SessionSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "session_success",
					Help:        "Indicates whether the entire restore session was succeeded or not",
					ConstLabels: labels,
				},
			),
			SessionDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "session_duration_seconds",
					Help:        "Indicates the total time taken to complete the entire restore session",
					ConstLabels: labels,
				},
			),
			TargetCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "target_count_total",
					Help:        "Indicates the total number of targets that was restored in this restore session",
					ConstLabels: labels,
				},
			),
		},
	}
}

func newRestoreTargetMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreTargetMetrics: &RestoreTargetMetrics{
			TargetRestoreSucceeded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "target_success",
					Help:        "Indicates whether the restore for a target has succeeded or not",
					ConstLabels: labels,
				},
			),
			HostCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "target_host_count_total",
					Help:        "Indicates the total number of hosts that was restored for this restore target",
					ConstLabels: labels,
				},
			),
		},
	}
}

func legacyRestoreTargetMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreTargetMetrics: &RestoreTargetMetrics{
			TargetRestoreSucceeded: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "target_success",
					Help:        "Indicates whether the restore for a target has succeeded or not",
					ConstLabels: labels,
				},
			),
			HostCount: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "target_host_count_total",
					Help:        "Indicates the total number of hosts that was restored for this restore target",
					ConstLabels: labels,
				},
			),
		},
	}
}

func newRestoreHostMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreHostMetrics: &RestoreHostMetrics{
			RestoreSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "host_restore_success",
					Help:        "Indicates whether the restore process was succeeded for a host",
					ConstLabels: labels,
				},
			),
			RestoreDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash_appscode_com",
					Subsystem:   "restoresession",
					Name:        "host_restore_duration_seconds",
					Help:        "Indicates the total time taken to complete the restore process for a host",
					ConstLabels: labels,
				},
			),
		},
	}
}

func legacyRestoreHostMetrics(labels prometheus.Labels) *RestoreMetrics {
	return &RestoreMetrics{
		RestoreHostMetrics: &RestoreHostMetrics{
			RestoreSuccess: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "host_restore_success",
					Help:        "Indicates whether the restore process was succeeded for a host",
					ConstLabels: labels,
				},
			),
			RestoreDuration: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "stash",
					Subsystem:   "restore",
					Name:        "host_restore_duration_seconds",
					Help:        "Indicates the total time taken to complete the restore process for a host",
					ConstLabels: labels,
				},
			),
		},
	}
}

// SendRestoreSessionMetrics send restore session related metrics to the Pushgateway
func (metricOpt *MetricsOptions) SendRestoreSessionMetrics(inv invoker.RestoreInvoker) error {
	// create metric registry
	registry := prometheus.NewRegistry()

	// generate metrics labels
	labels, err := restoreInvokerLabels(inv, metricOpt.Labels)
	if err != nil {
		return err
	}

	err = exportRestoreSessionMetrics(labels, inv, registry)
	if err != nil {
		return err
	}

	err = exportRestoreSessionLegacyMetrics(labels, inv, registry)
	if err != nil {
		return err
	}

	// send metrics to the pushgateway
	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func exportRestoreSessionMetrics(labels prometheus.Labels, inv invoker.RestoreInvoker, registry *prometheus.Registry) error {
	metrics := newRestoreSessionMetrics(labels)
	return setRestoreSessionMetrics(metrics, registry, inv)
}

func exportRestoreSessionLegacyMetrics(labels prometheus.Labels, inv invoker.RestoreInvoker, registry *prometheus.Registry) error {
	metrics := legacyRestoreSessionMetrics(labels)
	return setRestoreSessionMetrics(metrics, registry, inv)
}

func setRestoreSessionMetrics(metrics *RestoreMetrics, registry *prometheus.Registry, inv invoker.RestoreInvoker) error {
	if inv.GetStatus().Phase == api_v1beta1.RestoreSucceeded {
		// mark the entire restore session as succeeded
		metrics.RestoreSessionMetrics.SessionSuccess.Set(1)

		// set total time taken to complete the restore session
		duration, err := time.ParseDuration(inv.GetStatus().SessionDuration)
		if err != nil {
			return err
		}
		metrics.RestoreSessionMetrics.SessionDuration.Set(duration.Seconds())

		// set total number of target that was restored in this restore session
		metrics.RestoreSessionMetrics.TargetCount.Set(float64(len(inv.GetStatus().TargetStatus)))

		// register metrics to the registry
		registry.MustRegister(
			metrics.RestoreSessionMetrics.SessionSuccess,
			metrics.RestoreSessionMetrics.SessionDuration,
			metrics.RestoreSessionMetrics.TargetCount,
		)
	} else {
		// mark entire restore session as failed
		metrics.RestoreSessionMetrics.SessionSuccess.Set(0)
		registry.MustRegister(metrics.RestoreSessionMetrics.SessionSuccess)
	}

	return nil
}

// SendRestoreTargetMetrics send restore target related metrics to the Pushgateway
func (metricOpt *MetricsOptions) SendRestoreTargetMetrics(config *rest.Config, i invoker.RestoreInvoker, targetRef api_v1beta1.TargetRef) error {
	// create metric registry
	registry := prometheus.NewRegistry()

	// generate metrics labels
	labels, err := restoreInvokerLabels(i, metricOpt.Labels)
	if err != nil {
		return err
	}
	// generate target related labels
	targetLabels, err := targetLabels(config, targetRef, i.GetObjectMeta().Namespace)
	if err != nil {
		return err
	}
	labels = upsertLabel(labels, targetLabels)

	// only send the metric of the target specified by targetRef
	for _, targetStatus := range i.GetStatus().TargetStatus {
		if invoker.TargetMatched(targetStatus.Ref, targetRef) {

			exportRestoreTargetMetrics(labels, registry, targetStatus)
			exportRestoreTargetLegacyMetrics(labels, registry, targetStatus)

			// send metrics to the pushgateway
			return metricOpt.sendMetrics(registry, metricOpt.JobName)
		}
	}
	return nil
}

func exportRestoreTargetMetrics(labels prometheus.Labels, registry *prometheus.Registry, targetStatus api_v1beta1.RestoreMemberStatus) {
	metrics := newRestoreTargetMetrics(labels)
	setRestoreTargetMetrics(metrics, registry, targetStatus)
}

func exportRestoreTargetLegacyMetrics(labels prometheus.Labels, registry *prometheus.Registry, targetStatus api_v1beta1.RestoreMemberStatus) {
	metrics := legacyRestoreTargetMetrics(labels)
	setRestoreTargetMetrics(metrics, registry, targetStatus)
}

func setRestoreTargetMetrics(metrics *RestoreMetrics, registry *prometheus.Registry, targetStatus api_v1beta1.RestoreMemberStatus) {
	if targetStatus.Phase == api_v1beta1.TargetRestoreSucceeded {
		// mark entire restore target as succeeded
		metrics.RestoreTargetMetrics.TargetRestoreSucceeded.Set(1)

		// set total number of host that was restored in this restore session
		if targetStatus.TotalHosts != nil {
			metrics.RestoreTargetMetrics.HostCount.Set(float64(*targetStatus.TotalHosts))
		}

		// register metrics to the registry
		registry.MustRegister(
			metrics.RestoreTargetMetrics.TargetRestoreSucceeded,
			metrics.RestoreTargetMetrics.HostCount,
		)
	} else {
		// mark entire restore target as failed
		metrics.RestoreTargetMetrics.TargetRestoreSucceeded.Set(0)
		registry.MustRegister(metrics.RestoreTargetMetrics.TargetRestoreSucceeded)
	}
}

// SendRestoreHostMetrics send restore metrics for individual hosts to the Pushgateway
func (metricOpt *MetricsOptions) SendRestoreHostMetrics(config *rest.Config, i invoker.RestoreInvoker, targetRef api_v1beta1.TargetRef, restoreOutput *restic.RestoreOutput) error {
	if restoreOutput == nil {
		return fmt.Errorf("invalid restore output. Restore output shouldn't be nil")
	}

	// create metric registry
	registry := prometheus.NewRegistry()

	// generate restore session related labels
	labels, err := restoreInvokerLabels(i, metricOpt.Labels)
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
	for _, hostStats := range restoreOutput.RestoreTargetStatus.Stats {
		// add host name as label
		hostLabel := map[string]string{
			MetricLabelHostname: hostStats.Hostname,
		}

		metricLabels := upsertLabel(labels, hostLabel)

		err = exportRestoreHostMetrics(metricLabels, registry, hostStats)
		if err != nil {
			return err
		}

		err = exportRestoreHostLegacyMetrics(metricLabels, registry, hostStats)
		if err != nil {
			return err
		}
	}

	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func exportRestoreHostMetrics(labels prometheus.Labels, registry *prometheus.Registry, hostStats api_v1beta1.HostRestoreStats) error {
	metrics := newRestoreHostMetrics(labels)
	return setRestoreHostMetrics(metrics, registry, hostStats)
}

func exportRestoreHostLegacyMetrics(labels prometheus.Labels, registry *prometheus.Registry, hostStats api_v1beta1.HostRestoreStats) error {
	metrics := legacyRestoreHostMetrics(labels)
	return setRestoreHostMetrics(metrics, registry, hostStats)
}

func setRestoreHostMetrics(metrics *RestoreMetrics, registry *prometheus.Registry, hostStats api_v1beta1.HostRestoreStats) error {
	if hostStats.Phase == api_v1beta1.HostRestoreSucceeded {
		metrics.RestoreHostMetrics.RestoreSuccess.Set(1)

		// set the time that has been taken to restore the host
		duration, err := time.ParseDuration(hostStats.Duration)
		if err != nil {
			return err
		}
		metrics.RestoreHostMetrics.RestoreDuration.Set(duration.Seconds())

		registry.MustRegister(
			metrics.RestoreHostMetrics.RestoreSuccess,
			metrics.RestoreHostMetrics.RestoreDuration,
		)
	} else {
		// mark the host restore as failure
		metrics.RestoreHostMetrics.RestoreSuccess.Set(0)
		registry.MustRegister(
			metrics.RestoreHostMetrics.RestoreSuccess,
		)
	}
	return nil
}

// nolint:unparam
func restoreInvokerLabels(inv invoker.RestoreInvoker, userProvidedLabels []string) (prometheus.Labels, error) {
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

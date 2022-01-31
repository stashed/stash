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
	"context"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

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

// SendRepositoryMetrics send backup session related metrics to the Pushgateway
func (metricOpt *MetricsOptions) SendRepositoryMetrics(config *rest.Config, i invoker.BackupInvoker, repoStats restic.RepositoryStats) error {
	// create metric registry
	registry := prometheus.NewRegistry()

	// generate backup invoker labels
	labels, err := backupInvokerLabels(i, metricOpt.Labels)
	if err != nil {
		return err
	}

	repoMetricLabels, err := repoMetricLabels(config, i, metricOpt.Labels)
	if err != nil {
		return err
	}

	// create repository metrics
	repoMetrics := newRepositoryMetrics(upsertLabel(labels, repoMetricLabels))
	err = repoMetrics.setValues(repoStats)
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
	// send metrics to the pushgateway
	return metricOpt.sendMetrics(registry, metricOpt.JobName)
}

func repoMetricLabels(clientConfig *rest.Config, i invoker.BackupInvoker, userProvidedLabels []string) (prometheus.Labels, error) {
	// add user provided labels
	promLabels := parseUserProvidedLabels(userProvidedLabels)

	// insert repository information as label
	stashClient, err := cs.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	repository, err := stashClient.StashV1alpha1().Repositories(i.GetRepoRef().Namespace).Get(context.TODO(), i.GetRepoRef().Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	provider, err := repository.Spec.Backend.Provider()
	if err != nil {
		return nil, err
	}
	bucket, err := repository.Spec.Backend.Container()
	if err != nil {
		return nil, err
	}
	prefix, err := repository.Spec.Backend.Prefix()
	if err != nil {
		return nil, err
	}

	promLabels[MetricsLabelName] = repository.Name
	promLabels[MetricsLabelNamespace] = repository.Namespace
	promLabels[MetricsLabelBackend] = provider
	if bucket != "" {
		promLabels[MetricsLabelBucket] = bucket
	}
	if prefix != "" {
		promLabels[MetricsLabelPrefix] = prefix
	}
	return promLabels, nil
}

func (repoMetrics *RepositoryMetrics) setValues(repoStats restic.RepositoryStats) error {
	// set repository metrics values
	if repoStats.Integrity != nil && *repoStats.Integrity {
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

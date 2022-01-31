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
	"fmt"
	"path/filepath"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

const (
	MetricsLabelDriver     = "driver"
	MetricsLabelKind       = "kind"
	MetricsLabelAppGroup   = "group"
	MetricsLabelName       = "name"
	MetricsLabelNamespace  = "namespace"
	MetricsLabelRepository = "repository"
	MetricsLabelBackend    = "backend"
	MetricsLabelBucket     = "bucket"
	MetricsLabelPrefix     = "prefix"
	MetricLabelInvokerKind = "invoker_kind"
	MetricLabelInvokerName = "invoker_name"
	MetricLabelHostname    = "hostname"
)

type MetricsOptions struct {
	Enabled        bool
	PushgatewayURL string
	MetricFileDir  string
	Labels         []string
	JobName        string
}

var (
	pushgatewayURL string
)

const (
	PushgatewayLocalURL = "http://localhost:56789"
)

func SetPushgatewayURL(url string) {
	pushgatewayURL = PushgatewayLocalURL
	if url != "" {
		pushgatewayURL = url
	}
}

func GetPushgatewayURL() string {
	return pushgatewayURL
}

func (metricOpt *MetricsOptions) sendMetrics(registry *prometheus.Registry, jobName string) error {
	// if Pushgateway URL is provided, then push the metrics to Pushgateway
	if metricOpt.PushgatewayURL != "" {
		pusher := push.New(metricOpt.PushgatewayURL, jobName)
		err := pusher.Gatherer(registry).Add()
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

func upsertLabel(original, new map[string]string) map[string]string {
	labels := make(map[string]string)
	// copy old original labels
	for k, v := range original {
		labels[k] = v
	}
	// insert new labels
	for k, v := range new {
		labels[k] = v
	}
	return labels
}

// targetLabels returns backup/restore target specific labels
func targetLabels(config *rest.Config, target api_v1beta1.TargetRef, namespace string) (map[string]string, error) {

	labels := make(map[string]string)
	switch target.Kind {
	case apis.KindAppBinding:
		appGroup, appKind, err := getAppGroupKind(config, target.Name, namespace)
		// For PerconaXtradDB cluster restore, AppBinding will not exist during restore.
		// In this case, we can not add AppBinding specific labels.
		if err == nil {
			labels[MetricsLabelKind] = appKind
			labels[MetricsLabelAppGroup] = appGroup
		} else if !kerr.IsNotFound(err) {
			return nil, err
		}
	default:
		labels[MetricsLabelKind] = target.Kind
		gv, err := schema.ParseGroupVersion(target.APIVersion)
		if err != nil {
			return nil, err
		}
		labels[MetricsLabelAppGroup] = gv.Group
	}
	labels[MetricsLabelName] = target.Name
	return labels, nil
}

// volumeSnapshotterLabels returns volume snapshot specific labels
func volumeSnapshotterLabels() map[string]string {
	return map[string]string{
		MetricsLabelDriver:   string(api_v1beta1.VolumeSnapshotter),
		MetricsLabelKind:     apis.KindPersistentVolumeClaim,
		MetricsLabelAppGroup: core.GroupName,
	}
}

func getAppGroupKind(clientConfig *rest.Config, name, namespace string) (string, string, error) {
	appClient, err := appcatalog_cs.NewForConfig(clientConfig)
	if err != nil {
		return "", "", err
	}
	appbinding, err := appClient.AppcatalogV1alpha1().AppBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	// if app type is provided then use app group and app resource name.
	// otherwise, default to AppBinding's group,resources name
	targetAppGroup, targetAppResource := appbinding.AppGroupResource()
	if targetAppGroup == "" && targetAppResource == "" {
		targetAppGroup = appbinding.GroupVersionKind().Group
		targetAppResource = appcatalog.ResourceApps
	}
	return targetAppGroup, targetAppResource, nil
}

// parseUserProvidedLabels parses the labels provided by user as an array of key-value pair
// and returns labels in Prometheus labels format
func parseUserProvidedLabels(userLabels []string) prometheus.Labels {
	labels := prometheus.Labels{}
	for _, v := range userLabels {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	return labels
}

func convertSizeToBytes(dataSize string) (float64, error) {
	var size float64

	switch {
	case strings.HasSuffix(dataSize, "TiB"):
		_, err := fmt.Sscanf(dataSize, "%f TiB", &size)
		if err != nil {
			return 0, err
		}
		return size * (1 << 40), nil
	case strings.HasSuffix(dataSize, "GiB"):
		_, err := fmt.Sscanf(dataSize, "%f GiB", &size)
		if err != nil {
			return 0, err
		}
		return size * (1 << 30), nil
	case strings.HasSuffix(dataSize, "MiB"):
		_, err := fmt.Sscanf(dataSize, "%f MiB", &size)
		if err != nil {
			return 0, err
		}
		return size * (1 << 20), nil
	case strings.HasSuffix(dataSize, "KiB"):
		_, err := fmt.Sscanf(dataSize, "%f KiB", &size)
		if err != nil {
			return 0, err
		}
		return size * (1 << 10), nil
	default:
		_, err := fmt.Sscanf(dataSize, "%f B", &size)
		if err != nil {
			return 0, err
		}
		return size, nil

	}
}

func convertTimeToSeconds(processingTime string) (uint64, error) {
	var h, m, s uint64
	parts := strings.Split(processingTime, ":")
	if len(parts) == 3 {
		_, err := fmt.Sscanf(processingTime, "%d:%d:%d", &h, &m, &s)
		if err != nil {
			return 0, err
		}
	} else if len(parts) == 2 {
		_, err := fmt.Sscanf(processingTime, "%d:%d", &m, &s)
		if err != nil {
			return 0, err
		}
	} else {
		_, err := fmt.Sscanf(processingTime, "%d", &s)
		if err != nil {
			return 0, err
		}
	}

	return h*3600 + m*60 + s, nil
}

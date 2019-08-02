package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	store "kmodules.xyz/objectstore-api/api/v1"
	v1 "kmodules.xyz/offshoot-api/api/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
)

var (
	ServiceName string
)

const (
	CallerWebhook    = "webhook"
	CallerController = "controller"
)

type RepoLabelData struct {
	WorkloadKind string
	WorkloadName string
	PodName      string
	NodeName     string
}

// GetHostName returns hostname for a target
func GetHostName(target interface{}) (string, error) {
	// target nil for cluster backup
	var targetRef api_v1beta1.TargetRef
	if target == nil {
		return "host-0", nil
	}

	// read targetRef field from BackupTarget or RestoreTarget
	switch target.(type) {
	case *api_v1beta1.BackupTarget:
		t := target.(*api_v1beta1.BackupTarget)
		if t == nil {
			return "host-0", nil
		}
		targetRef = t.Ref
	case *api_v1beta1.RestoreTarget:
		t := target.(*api_v1beta1.RestoreTarget)
		if t == nil {
			return "host-0", nil
		}

		// if replicas or volumeClaimTemplate is specified then  restore is done via job.
		// in this case, we need to know the ordinal to use as host suffix.
		// stash operator sets desired ordinal as 'POD_ORDINAL' env while creating the job.
		if t.Replicas != nil || len(t.VolumeClaimTemplates) != 0 {
			if os.Getenv(KeyPodOrdinal) != "" {
				return "host-" + os.Getenv(KeyPodOrdinal), nil
			}
			return "", fmt.Errorf("'target.replicas' or 'target.volumeClaimTemplate' has been specified in RestoreSession" +
				" but 'POD_ORDINAL' env not found")
		}
		targetRef = t.Ref
	}

	// backup/restore is running through sidecar/init-container. identify hostname for them.
	switch targetRef.Kind {
	case apis.KindStatefulSet:
		// for StatefulSet, host name is 'host-<pod ordinal>'. stash operator set pod's name as 'POD_NAME' env
		// in the sidecar/init-container through downward api. we have to parse the pod name to get ordinal.
		podName := os.Getenv(KeyPodName)
		if podName == "" {
			return "", fmt.Errorf("missing 'POD_NAME' env in StatefulSet: %s", apis.KindStatefulSet)
		}
		podInfo := strings.Split(podName, "-")
		podOrdinal := podInfo[len(podInfo)-1]
		return "host-" + podOrdinal, nil
	case apis.KindDaemonSet:
		// for DaemonSet, host name is the node name. stash operator set the respective node name as 'NODE_NAME' env
		// in the sidecar/init-container through downward api.
		nodeName := os.Getenv(KeyNodeName)
		if nodeName == "" {
			return "", fmt.Errorf("missing 'NODE_NAME' env for DaemonSet: %s", apis.KindDaemonSet)
		}
		return nodeName, nil
	default:
		return "host-0", nil
	}
}

func PushgatewayURL() string {
	// called by operator, returning its own namespace. Since pushgateway runs as a side-car with operator, this works!
	return fmt.Sprintf("http://%s.%s.svc:56789", ServiceName, meta.Namespace())
}

func BackupModel(kind string) string {
	switch kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindStatefulSet, apis.KindDaemonSet, apis.KindDeploymentConfig:
		return ModelSidecar
	default:
		return ModelCronJob
	}
}

func GetRepoNameAndSnapshotID(snapshotName string) (repoName, snapshotId string, err error) {
	if len(snapshotName) < 9 {
		err = errors.New("invalid snapshot name")
		return
	}
	tokens := strings.SplitN(snapshotName, "-", -1)
	if len(tokens) < 2 {
		err = errors.New("invalid snapshot name")
		return
	}
	snapshotId = tokens[len(tokens)-1]
	if len(snapshotId) != SnapshotIDLength {
		err = errors.New("invalid snapshot name")
		return
	}

	repoName = strings.TrimSuffix(snapshotName, "-"+snapshotId)
	return
}

func FixBackendPrefix(backend *store.Backend, autoPrefix string) *store.Backend {
	if backend.Local != nil {
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, autoPrefix)
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, "/")
	} else if backend.S3 != nil {
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, autoPrefix)
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, "/")
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket)
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, "/")
	} else if backend.GCS != nil {
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, autoPrefix)
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, "/")
	} else if backend.Azure != nil {
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, autoPrefix)
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, "/")
	} else if backend.Swift != nil {
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, autoPrefix)
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, "/")
	} else if backend.B2 != nil {
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, autoPrefix)
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, "/")
	}
	return backend
}

func GetEndpoint(backend *store.Backend) string {
	if backend.S3 != nil {
		return backend.S3.Endpoint
	}
	return ""
}

// TODO: move to store
func GetBucketAndPrefix(backend *store.Backend) (string, string, error) {
	if backend.Local != nil {
		return "", filepath.Join(backend.Local.MountPath, strings.TrimPrefix(backend.Local.SubPath, "/")), nil
	} else if backend.S3 != nil {
		return backend.S3.Bucket, strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket+"/"), nil
	} else if backend.GCS != nil {
		return backend.GCS.Bucket, backend.GCS.Prefix, nil
	} else if backend.Azure != nil {
		return backend.Azure.Container, backend.Azure.Prefix, nil
	} else if backend.Swift != nil {
		return backend.Swift.Container, backend.Swift.Prefix, nil
	} else if backend.Rest != nil {
		return "", "", nil
	}
	return "", "", errors.New("unknown backend type.")
}

// TODO: move to store
func GetProvider(backend store.Backend) (string, error) {
	if backend.Local != nil {
		return restic.ProviderLocal, nil
	} else if backend.S3 != nil {
		return restic.ProviderS3, nil
	} else if backend.GCS != nil {
		return restic.ProviderGCS, nil
	} else if backend.Azure != nil {
		return restic.ProviderAzure, nil
	} else if backend.Swift != nil {
		return restic.ProviderSwift, nil
	} else if backend.B2 != nil {
		return restic.ProviderB2, nil
	} else if backend.Rest != nil {
		return restic.ProviderRest, nil
	}
	return "", errors.New("unknown provider.")
}

func GetRestUrl(backend store.Backend) string {
	if backend.Rest != nil {
		return backend.Rest.URL
	}
	return ""
}

// TODO: move to store
// returns 0 if not specified
func GetMaxConnections(backend store.Backend) int {
	if backend.GCS != nil {
		return backend.GCS.MaxConnections
	} else if backend.Azure != nil {
		return backend.Azure.MaxConnections
	} else if backend.B2 != nil {
		return backend.B2.MaxConnections
	}
	return 0
}

func ExtractDataFromRepositoryLabel(labels map[string]string) (data RepoLabelData, err error) {
	var ok bool
	data.WorkloadKind, ok = labels["workload-kind"]
	if !ok {
		return data, errors.New("workload-kind not found in repository labels")
	}

	data.WorkloadName, ok = labels["workload-name"]
	if !ok {
		return data, errors.New("workload-name not found in repository labels")
	}

	data.PodName, ok = labels["pod-name"]
	if !ok {
		data.PodName = ""
	}

	data.NodeName, ok = labels["node-name"]
	if !ok {
		data.NodeName = ""
	}
	return data, nil
}

func AttachLocalBackend(podSpec core.PodSpec, localSpec store.LocalSpec) core.PodSpec {
	volume, mount := localSpec.ToVolumeAndMount(LocalVolumeName)
	podSpec.Volumes = core_util.UpsertVolume(podSpec.Volumes, volume)
	for i := range podSpec.InitContainers {
		podSpec.InitContainers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.InitContainers[i].VolumeMounts, mount)
	}
	for i := range podSpec.Containers {
		podSpec.Containers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.Containers[i].VolumeMounts, mount)
	}
	return podSpec
}

func AttachPVC(podSpec core.PodSpec, volumes []core.Volume, volumeMounts []core.VolumeMount) core.PodSpec {
	podSpec.Volumes = core_util.UpsertVolume(podSpec.Volumes, volumes...)
	for i := range podSpec.InitContainers {
		podSpec.InitContainers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.InitContainers[i].VolumeMounts, volumeMounts...)
	}
	for i := range podSpec.Containers {
		podSpec.Containers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.Containers[i].VolumeMounts, volumeMounts...)
	}
	return podSpec
}

func NiceSettingsFromEnv() (*v1.NiceSettings, error) {
	var settings *v1.NiceSettings
	if v, ok := os.LookupEnv(apis.NiceAdjustment); ok {
		vi, err := ParseInt32P(v)
		if err != nil {
			return nil, err
		}
		settings = &v1.NiceSettings{
			Adjustment: vi,
		}
	}
	return settings, nil
}

func IONiceSettingsFromEnv() (*v1.IONiceSettings, error) {
	var settings *v1.IONiceSettings
	if v, ok := os.LookupEnv(apis.IONiceClass); ok {
		vi, err := ParseInt32P(v)
		if err != nil {
			return nil, err
		}
		settings = &v1.IONiceSettings{
			Class: vi,
		}
	}
	if v, ok := os.LookupEnv(apis.IONiceClassData); ok {
		vi, err := ParseInt32P(v)
		if err != nil {
			return nil, err
		}
		if settings == nil {
			settings = &v1.IONiceSettings{}
		}
		settings.ClassData = vi
	}
	return settings, nil
}

func ParseInt32P(v string) (*int32, error) {
	vi, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}
	return types.Int32P(int32(vi)), nil
}

type WorkloadClients struct {
	KubeClient       kubernetes.Interface
	OcClient         oc_cs.Interface
	StashClient      cs.Interface
	CRDClient        crd_cs.ApiextensionsV1beta1Interface
	AppCatalogClient appcatalog_cs.Interface
}

func (wc *WorkloadClients) IsTargetExist(target api_v1beta1.TargetRef, namespace string) bool {
	switch target.Kind {
	case apis.KindDeployment:
		if _, err := wc.KubeClient.AppsV1().Deployments(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindDaemonSet:
		if _, err := wc.KubeClient.AppsV1().DaemonSets(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindStatefulSet:
		if _, err := wc.KubeClient.AppsV1().StatefulSets(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindReplicationController:
		if _, err := wc.KubeClient.CoreV1().ReplicationControllers(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindReplicaSet:
		if _, err := wc.KubeClient.AppsV1().StatefulSets(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindDeploymentConfig:
		if wc.OcClient != nil {
			if _, err := wc.OcClient.AppsV1().DeploymentConfigs(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
				return true
			}
		}
	case apis.KindPersistentVolumeClaim:
		if _, err := wc.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	case apis.KindAppBinding:
		if _, err := wc.AppCatalogClient.AppcatalogV1alpha1().AppBindings(namespace).Get(target.Name, metav1.GetOptions{}); err == nil {
			return true
		}
	}
	return false
}

// CreateBatchPVC creates a batch of PVCs whose definitions has been provided in pvcList argument
func CreateBatchPVC(kubeClient kubernetes.Interface, namespace string, pvcList []core.PersistentVolumeClaim) error {
	for _, pvc := range pvcList {
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(&pvc)
		if err != nil {
			return err
		}
	}
	return nil
}

// PVCListToVolumes return a list of volumes to mount in pod for a list of PVCs
func PVCListToVolumes(pvcList []core.PersistentVolumeClaim, ordinal int32) []core.Volume {
	volList := make([]core.Volume, 0)
	var volName string
	for _, pvc := range pvcList {
		volName = strings.TrimSuffix(pvc.Name, fmt.Sprintf("-%d", ordinal))
		volList = append(volList, core.Volume{
			Name: volName,
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})
	}
	return volList
}

func HasStashContainer(w *wapi.Workload) bool {
	return HasStashSidecar(w.Spec.Template.Spec.Containers) || HasStashInitContainer(w.Spec.Template.Spec.InitContainers)
}

func HasStashSidecar(containers []core.Container) bool {
	// check if the workload has stash sidecar container
	for _, c := range containers {
		if c.Name == StashContainer {
			return true
		}
	}
	return false
}

func HasStashInitContainer(containers []core.Container) bool {
	// check if the workload has stash init-container
	for _, c := range containers {
		if c.Name == StashInitContainer {
			return true
		}
	}
	return false
}

/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"context"
	"fmt"
	"os"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/metrics"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	store "kmodules.xyz/objectstore-api/api/v1"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
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
	if target == nil {
		return apis.DefaultHost, nil
	}

	var alias string
	var targetRef api_v1beta1.TargetRef

	// read targetRef field from BackupTarget or RestoreTarget
	switch t := target.(type) {
	case *api_v1beta1.BackupTarget:
		if t == nil {
			return apis.DefaultHost, nil
		}
		alias = t.Alias
		targetRef = t.Ref
	case *api_v1beta1.RestoreTarget:
		if t == nil {
			return apis.DefaultHost, nil
		}
		alias = t.Alias
		targetRef = t.Ref

		// if volumeClaimTemplate is specified then  restore is done via job.
		// in this case, we have to add pod ordinal as host suffix for StatefulSet.
		// stash operator sets desired ordinal as 'POD_ORDINAL' env while creating the job.
		if len(t.VolumeClaimTemplates) != 0 {
			if t.Replicas != nil {
				// restoring the volumes of a StatefulSet. so, add pod ordinal suffix.
				if os.Getenv(apis.KeyPodOrdinal) != "" {
					if alias != "" {
						return fmt.Sprintf("%s-%s", alias, os.Getenv(apis.KeyPodOrdinal)), nil
					}
					return "host-" + os.Getenv(apis.KeyPodOrdinal), nil
				}
				return "", fmt.Errorf("'target.volumeClaimTemplate' has been specified in the restore invoker" +
					" but 'POD_ORDINAL' env not found")
			}
			// restoring volume of the other workloads. in this case, we don't have to add the pod ordinal suffix.
			if alias != "" {
				return alias, nil
			}
			return apis.DefaultHost, nil
		}
	}

	// backup/restore is running through sidecar/init-container. identify hostname for them.
	switch targetRef.Kind {
	case apis.KindStatefulSet:
		// for StatefulSet, host name is 'host-<pod ordinal>'. stash operator set pod's name as 'POD_NAME' env
		// in the sidecar/init-container through downward api. we have to parse the pod name to get ordinal.
		podName := meta_util.PodName()
		if podName == "" {
			return "", fmt.Errorf("missing 'POD_NAME' env in StatefulSet: %s", apis.KindStatefulSet)
		}
		podInfo := strings.Split(podName, "-")
		podOrdinal := podInfo[len(podInfo)-1]
		if alias != "" {
			return fmt.Sprintf("%s-%s", alias, podOrdinal), nil
		}
		return "host-" + podOrdinal, nil
	case apis.KindDaemonSet:
		// for DaemonSet, host name is the node name. stash operator set the respective node name as 'NODE_NAME' env
		// in the sidecar/init-container through downward api.
		nodeName := os.Getenv(apis.KeyNodeName)
		if nodeName == "" {
			return "", fmt.Errorf("missing 'NODE_NAME' env for DaemonSet: %s", apis.KindDaemonSet)
		}
		if alias != "" {
			return fmt.Sprintf("%s-%s", alias, nodeName), nil
		}
		return nodeName, nil
	default:
		if alias != "" {
			return alias, nil
		}
		return apis.DefaultHost, nil
	}
}

func BackupModel(kind, taskName string) string {
	if taskName == "" && isWorkload(kind) {
		return apis.ModelSidecar
	}
	return apis.ModelCronJob
}

func isWorkload(kind string) bool {
	return kind == apis.KindDeployment ||
		kind == apis.KindStatefulSet ||
		kind == apis.KindDaemonSet ||
		kind == apis.KindDeploymentConfig
}

func RestoreModel(kind, taskName string) string {
	return BackupModel(kind, taskName)
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
	if len(snapshotId) != apis.SnapshotIDLength {
		err = errors.New("invalid snapshot name")
		return
	}

	repoName = strings.TrimSuffix(snapshotName, "-"+snapshotId)
	return
}

func FixBackendPrefix(backend *store.Backend, autoPrefix string) *store.Backend {
	if backend.S3 != nil {
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

// TODO: move to store
func GetBucketAndPrefix(backend *store.Backend) (string, string, error) {
	if backend.S3 != nil {
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

func AttachPVC(podSpec core.PodSpec, volumes []core.Volume, volumeMounts []core.VolumeMount) core.PodSpec {
	if len(volumeMounts) > 0 {
		podSpec.Volumes = core_util.UpsertVolume(podSpec.Volumes, volumes...)
		for i := range podSpec.InitContainers {
			podSpec.InitContainers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.InitContainers[i].VolumeMounts, volumeMounts...)
		}
		for i := range podSpec.Containers {
			podSpec.Containers[i].VolumeMounts = core_util.UpsertVolumeMount(podSpec.Containers[i].VolumeMounts, volumeMounts...)
		}
	}
	return podSpec
}

func IsTargetExist(config *rest.Config, target api_v1beta1.TargetRef) (bool, error) {
	if target.Kind == api_v1beta1.TargetKindEmpty {
		return true, nil
	}

	mapping, err := getRESTMapping(config, target)
	if err != nil {
		return false, err
	}

	di, err := dynamic.NewForConfig(config)
	if err != nil {
		return false, err
	}

	var ri dynamic.ResourceInterface
	ri = di.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ri = di.Resource(mapping.Resource).Namespace(target.Namespace)
	}

	_, err = ri.Get(context.TODO(), target.Name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getRESTMapping(config *rest.Config, target api_v1beta1.TargetRef) (*meta.RESTMapping, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	apiResources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return nil, err
	}

	gv, err := schema.ParseGroupVersion(target.APIVersion)
	if err != nil {
		return nil, err
	}
	gvk := gv.WithKind(target.Kind)

	mapper := restmapper.NewDiscoveryRESTMapper(apiResources)
	return mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
}

// CreateBatchPVC creates a batch of PVCs whose definitions has been provided in pvcList argument
func CreateBatchPVC(kubeClient kubernetes.Interface, namespace string, pvcList []core.PersistentVolumeClaim) error {
	for _, pvc := range pvcList {
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), &pvc, metav1.CreateOptions{})
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
		if c.Name == apis.StashContainer {
			return true
		}
	}
	return false
}

func HasStashInitContainer(containers []core.Container) bool {
	// check if the workload has stash init-container
	for _, c := range containers {
		if c.Name == apis.StashInitContainer {
			return true
		}
	}
	return false
}

// UpsertInterimVolume create a PVC according to InterimVolumeTemplate and attach it to the respective pod
func UpsertInterimVolume(kc kubernetes.Interface, podSpec core.PodSpec, interimVolumeTemplate *core.PersistentVolumeClaim, namespace string, owner *metav1.OwnerReference) (core.PodSpec, error) {
	// if no InterimVolumeTemplate is provided then nothing to do
	if interimVolumeTemplate == nil {
		return podSpec, nil
	}

	// Use owner name as prefix of the interim volume
	pvcMeta := metav1.ObjectMeta{
		Name:      meta_util.ValidNameWithPrefix(owner.Name, interimVolumeTemplate.Name),
		Namespace: namespace,
	}

	// create the interim pvc
	createdPVC, _, err := core_util.CreateOrPatchPVC(context.TODO(), kc, pvcMeta, func(in *core.PersistentVolumeClaim) *core.PersistentVolumeClaim {
		// Set BackupSession/RestoreSession as owner of the PVC so that it get deleted when the respective owner is deleted.
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		in.Spec = interimVolumeTemplate.Spec
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return podSpec, err
	}

	// Attach the PVC to the pod template
	volumes := []core.Volume{
		{
			Name: apis.StashInterimVolume,
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: createdPVC.Name,
				},
			},
		},
	}
	volumeMounts := []core.VolumeMount{
		{
			Name:      apis.StashInterimVolume,
			MountPath: apis.StashInterimVolumeMountPath,
		},
	}
	return AttachPVC(podSpec, volumes, volumeMounts), nil
}

func HookExecutorContainer(name string, shiblings []core.Container, invokerKind, invokerName string, target api_v1beta1.TargetRef) core.Container {
	hookExecutor := core.Container{
		Name:  name,
		Image: "${STASH_DOCKER_REGISTRY:=appscode}/${STASH_DOCKER_IMAGE:=stash}:${STASH_IMAGE_TAG:=latest}",
		Args: []string{
			"run-hook",
			"--backupsession=${BACKUP_SESSION:=}",
			"--invoker-kind=" + invokerKind,
			"--invoker-name=" + invokerName,
			"--target-kind=" + target.Kind,
			"--target-namespace=" + target.Namespace,
			"--target-name=" + target.Name,
			"--hook-type=${HOOK_TYPE:=}",
			"--hostname=${HOSTNAME:=}",
			"--output-dir=${outputDir:=}",
			"--metrics-enabled=true",
			fmt.Sprintf("--metrics-pushgateway-url=%s", metrics.GetPushgatewayURL()),
		},
		Env: []core.EnvVar{
			{
				Name: apis.KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}
	// now, upsert the volumeMounts of the sibling containers
	// multiple containers may have same volume mounted on different directory
	// in such cae, we will give priority to the last one.
	var mounts []core.VolumeMount
	for _, c := range shiblings {
		mounts = append(mounts, c.VolumeMounts...)
	}
	hookExecutor.VolumeMounts = core_util.UpsertVolumeMount(hookExecutor.VolumeMounts, mounts...)

	return hookExecutor
}

func OwnerWorkload(w *wapi.Workload) (*metav1.OwnerReference, error) {
	switch w.Kind {
	case apis.KindDeployment, apis.KindStatefulSet, apis.KindDaemonSet:
		return metav1.NewControllerRef(w, appsv1.SchemeGroupVersion.WithKind(w.Kind)), nil
	case apis.KindDeploymentConfig:
		return metav1.NewControllerRef(w, ocapps.GroupVersion.WithKind(w.Kind)), nil
	default:
		return nil, fmt.Errorf("failed to set workload as owner. Reason: unknown workload kind")
	}
}

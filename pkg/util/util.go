/*
Copyright The Stash Authors.

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

package util

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/pushgateway"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	store "kmodules.xyz/objectstore-api/api/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	prober "kmodules.xyz/prober/probe"
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
	var targetRef api_v1beta1.TargetRef
	if target == nil {
		return apis.DefaultHost, nil
	}

	// read targetRef field from BackupTarget or RestoreTarget
	switch t := target.(type) {
	case *api_v1beta1.BackupTarget:
		if t == nil {
			return apis.DefaultHost, nil
		}
		targetRef = t.Ref
	case *api_v1beta1.RestoreTarget:
		if t == nil {
			return apis.DefaultHost, nil
		}

		// if replicas or volumeClaimTemplate is specified then  restore is done via job.
		// in this case, we need to know the ordinal to use as host suffix.
		// stash operator sets desired ordinal as 'POD_ORDINAL' env while creating the job.
		if t.Replicas != nil || len(t.VolumeClaimTemplates) != 0 {
			if os.Getenv(apis.KeyPodOrdinal) != "" {
				return "host-" + os.Getenv(apis.KeyPodOrdinal), nil
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
		podName := os.Getenv(apis.KeyPodName)
		if podName == "" {
			return "", fmt.Errorf("missing 'POD_NAME' env in StatefulSet: %s", apis.KindStatefulSet)
		}
		podInfo := strings.Split(podName, "-")
		podOrdinal := podInfo[len(podInfo)-1]
		return "host-" + podOrdinal, nil
	case apis.KindDaemonSet:
		// for DaemonSet, host name is the node name. stash operator set the respective node name as 'NODE_NAME' env
		// in the sidecar/init-container through downward api.
		nodeName := os.Getenv(apis.KeyNodeName)
		if nodeName == "" {
			return "", fmt.Errorf("missing 'NODE_NAME' env for DaemonSet: %s", apis.KindDaemonSet)
		}
		return nodeName, nil
	default:
		return apis.DefaultHost, nil
	}
}

func GetRestoreHostName(stashClient cs.Interface, restoreSessionName, namespace string) (string, error) {
	restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), restoreSessionName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if restoreSession.Spec.Target != nil {
		return GetHostName(restoreSession.Spec.Target)
	}
	return apis.DefaultHost, nil
}

func BackupModel(kind string) string {
	switch kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindStatefulSet, apis.KindDaemonSet, apis.KindDeploymentConfig:
		return apis.ModelSidecar
	default:
		return apis.ModelCronJob
	}
}

func RestoreModel(kind string) string {
	return BackupModel(kind)
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
	volume, mount := localSpec.ToVolumeAndMount(apis.LocalVolumeName)
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

type WorkloadClients struct {
	KubeClient       kubernetes.Interface
	OcClient         oc_cs.Interface
	StashClient      cs.Interface
	CRDClient        crd_cs.Interface
	AppCatalogClient appcatalog_cs.Interface
}

func (wc *WorkloadClients) IsTargetExist(target api_v1beta1.TargetRef, namespace string) (bool, error) {
	var err error
	switch target.Kind {
	case apis.KindDeployment:
		if _, err = wc.KubeClient.AppsV1().Deployments(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindDaemonSet:
		if _, err = wc.KubeClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindStatefulSet:
		if _, err = wc.KubeClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindReplicationController:
		if _, err = wc.KubeClient.CoreV1().ReplicationControllers(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindReplicaSet:
		if _, err = wc.KubeClient.AppsV1().ReplicaSets(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindDeploymentConfig:
		if wc.OcClient != nil {
			if _, err = wc.OcClient.AppsV1().DeploymentConfigs(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
				return true, nil
			}
		}
	case apis.KindPersistentVolumeClaim:
		if _, err = wc.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	case apis.KindAppBinding:
		if _, err = wc.AppCatalogClient.AppcatalogV1alpha1().AppBindings(namespace).Get(context.TODO(), target.Name, metav1.GetOptions{}); err == nil {
			return true, nil
		}
	}
	if err != nil && !kerr.IsNotFound(err) {
		return false, err
	}
	return false, nil
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

// GetWorkloadReference return reference of the workload.
func GetWorkloadReference(w *wapi.Workload) (*core.ObjectReference, error) {
	ref, err := reference.GetReference(scheme.Scheme, w)
	if err != nil && err != reference.ErrNilObject {
		return &core.ObjectReference{
			Name:       w.Name,
			Namespace:  w.Namespace,
			APIVersion: w.APIVersion,
		}, nil
	}
	return ref, err
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

// xref: https://kubernetes.io/docs/reference/kubectl/overview/#resource-types
func ResourceKindShortForm(kind string) string {
	switch kind {
	case apis.KindDeployment:
		return "deploy"
	case apis.KindReplicationController:
		return "rc"
	case apis.KindDaemonSet:
		return "ds"
	case apis.KindStatefulSet:
		return "sts"
	case apis.KindPersistentVolumeClaim:
		return "pvc"
	case apis.KindPod:
		return "po"
	case apis.KindAppBinding:
		return "app"
	default:
		return strings.ToLower(kind)
	}
}

func ExecuteHook(config *rest.Config, hook interface{}, hookType, podName, namespace string) error {
	var hookErr error
	log.Infof("Executing %s hooks.........\n", hookType)

	switch h := hook.(type) {
	case *api_v1beta1.BackupHooks:
		if hookType == apis.PreBackupHook {
			hookErr = prober.RunProbe(config, h.PreBackup, podName, namespace)
		} else {
			err := prober.RunProbe(config, h.PostBackup, podName, namespace)
			if err != nil {
				hookErr = fmt.Errorf(err.Error() + ". Warning: The actual backup process may be succeeded." +
					" Hence, backup data might be present in the backend even if the overall BackupSession phase is 'Failed'")
			}
		}
	case *api_v1beta1.RestoreHooks:
		if hookType == apis.PreRestoreHook {
			hookErr = prober.RunProbe(config, h.PreRestore, podName, namespace)
		} else {
			err := prober.RunProbe(config, h.PostRestore, podName, namespace)
			if err != nil {
				hookErr = fmt.Errorf(err.Error() + ". Warning: The actual restore process may be succeeded." +
					" Hence, the restored data might be present in the target even if the overall RestoreSession phase is 'Failed'")
			}
		}
	default:
		return fmt.Errorf("unknown hook type")
	}

	if hookErr != nil {
		return hookErr
	}

	log.Infof("Successfully executed %s hook.\n", hookType)
	return nil
}

func HookExecutorContainer(name string, shiblings []core.Container, invokerType, invokerName, targetKind, targetName string) core.Container {
	hookExecutor := core.Container{
		Name:  name,
		Image: "${STASH_DOCKER_REGISTRY:=appscode}/${STASH_DOCKER_IMAGE:=stash}:${STASH_IMAGE_TAG:=latest}",
		Args: []string{
			"run-hook",
			"--backupsession=${BACKUP_SESSION:=}",
			"--restoresession=${RESTORE_SESSION:=}",
			"--invoker-type=" + invokerType,
			"--invoker-name=" + invokerName,
			"--target-kind=" + targetKind,
			"--target-name=" + targetName,
			"--hook-type=${HOOK_TYPE:=}",
			"--hostname=${HOSTNAME:=}",
			"--output-dir=${outputDir:=}",
			"--metrics-enabled=true",
			fmt.Sprintf("--metrics-pushgateway-url=%s", pushgateway.URL()),
			"--prom-job-name=${PROMETHEUS_JOB_NAME:=}",
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

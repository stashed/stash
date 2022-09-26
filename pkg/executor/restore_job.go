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

package executor

import (
	"fmt"
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolver"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/flags"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clientcmd"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

type RestoreJob struct {
	KubeClient        kubernetes.Interface
	StashClient       cs.Interface
	CatalogClient     appcatalog_cs.Interface
	RBACOptions       *rbac.Options
	ImagePullSecrets  []core.LocalObjectReference
	Invoker           invoker.RestoreInvoker
	Index             int
	Repository        *v1alpha1.Repository
	LicenseApiService string
	Image             docker.Docker
}

func (e *RestoreJob) Ensure() (runtime.Object, kutil.VerbType, error) {
	jobMeta := metav1.ObjectMeta{
		Name:      e.getName(),
		Namespace: e.Invoker.GetObjectMeta().Namespace,
		Labels:    e.Invoker.GetLabels(),
	}

	if err := e.RBACOptions.EnsureRestoreJobRBAC(); err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	if targetInfo.Target.VolumeClaimTemplates != nil {
		return e.ensureClaimedVolumesAndJob(jobMeta)
	}

	podSpec, err := e.resolveTask()
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return e.ensureJob(jobMeta, podSpec)
}

func (e *RestoreJob) ensureClaimedVolumesAndJob(jobMeta metav1.ObjectMeta) (runtime.Object, kutil.VerbType, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	replicas := int32(1)
	if targetInfo.Target.Replicas != nil {
		replicas = *targetInfo.Target.Replicas
	}

	// Now, we have to do the following for each replica:
	// 1. Create PVCs according to the template.
	// 2. Mount the PVCs to the restore job.
	// 3. Create the restore job to restore into the mounted volume.
	verb := kutil.VerbUnchanged
	for ordinal := int32(0); ordinal < replicas; ordinal++ {
		pvcList, err := e.ensureClaimedVolumes(ordinal)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		// add ordinal suffix to the job name so that multiple restore job can run concurrently
		rMeta := jobMeta.DeepCopy()
		rMeta.Name = fmt.Sprintf("%s-%d", jobMeta.Name, ordinal)

		podSpec, err := e.getPodTemplateForVolumes(pvcList, ordinal)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		_, v, err := e.ensureJob(*rMeta, podSpec)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		if v != kutil.VerbUnchanged {
			verb = v
		}

	}
	return nil, verb, nil
}

func (e *RestoreJob) ensureClaimedVolumes(ordinal int32) ([]core.PersistentVolumeClaim, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	r := resolver.VolumeTemplateOptions{
		Ordinal:         int(ordinal),
		VolumeTemplates: targetInfo.Target.VolumeClaimTemplates,
	}
	pvcList, err := r.Resolve()
	if err != nil {
		return nil, err
	}
	if err := util.CreateBatchPVC(e.KubeClient, e.Invoker.GetObjectMeta().Namespace, pvcList); err != nil {
		return nil, err
	}
	return pvcList, err
}

func (e *RestoreJob) getPodTemplateForVolumes(pvcList []core.PersistentVolumeClaim, ordinal int32) (core.PodSpec, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	// if restore process follows Function-Task model, then resolve the Functions and Task  for this host
	var podSpec core.PodSpec
	var err error
	if targetInfo.Task.Name != "" {
		podSpec, err = e.resolveTask()
		if err != nil {
			return core.PodSpec{}, err
		}
	} else {
		podSpec = e.volumeRestorerPodTemplate()
	}

	// mount the newly created PVCs into the job
	volumes := util.PVCListToVolumes(pvcList, ordinal)
	podSpec = util.AttachPVC(podSpec, volumes, targetInfo.Target.VolumeMounts)

	ordinalEnv := core.EnvVar{
		Name:  apis.KeyPodOrdinal,
		Value: fmt.Sprintf("%d", ordinal),
	}

	// insert POD_ORDINAL env in all init-containers.
	for i, c := range podSpec.InitContainers {
		podSpec.InitContainers[i].Env = core_util.UpsertEnvVars(c.Env, ordinalEnv)
	}

	// insert POD_ORDINAL env in all containers.
	for i, c := range podSpec.Containers {
		podSpec.Containers[i].Env = core_util.UpsertEnvVars(c.Env, ordinalEnv)
	}
	return podSpec, nil
}

func (e *RestoreJob) resolveTask() (core.PodSpec, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	r := resolver.TaskOptions{
		StashClient:       e.StashClient,
		CatalogClient:     e.CatalogClient,
		Repository:        e.Repository,
		Image:             e.Image,
		LicenseApiService: e.LicenseApiService,
		Restore: &resolver.RestoreOptions{
			Invoker:    e.Invoker,
			TargetInfo: targetInfo,
		},
	}
	podSpec, err := r.Resolve()
	if err != nil {
		return core.PodSpec{}, err
	}

	return util.UpsertInterimVolume(
		e.KubeClient,
		podSpec,
		targetInfo.InterimVolumeTemplate.ToCorePVC(),
		e.Invoker.GetObjectMeta().Namespace,
		e.Invoker.GetOwnerRef(),
	)
}

func (e *RestoreJob) ensureJob(jobMeta metav1.ObjectMeta, podSpec core.PodSpec) (runtime.Object, kutil.VerbType, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	runtimeSettings := targetInfo.RuntimeSettings

	job := jobOptions{
		kubeClient:         e.KubeClient,
		meta:               jobMeta,
		owner:              e.Invoker.GetOwnerRef(),
		podSpec:            podSpec,
		podLabels:          e.Invoker.GetLabels(),
		serviceAccountName: e.RBACOptions.GetServiceAccountName(),
		imagePullSecrets:   e.ImagePullSecrets,
		backOffLimit:       0,
	}
	if runtimeSettings.Pod != nil && runtimeSettings.Pod.PodAnnotations != nil {
		job.podAnnotations = runtimeSettings.Pod.PodAnnotations
	}
	return job.ensure()
}

func (e *RestoreJob) getName() string {
	return meta_util.ValidNameWithPrefixNSuffix(
		apis.PrefixStashRestore,
		strings.ReplaceAll(e.Invoker.GetObjectMeta().Name, ".", "-"),
		strconv.Itoa(e.Index),
	)
}

func (e *RestoreJob) volumeRestorerPodTemplate() core.PodSpec {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	container := core.Container{
		Name:  apis.StashContainer,
		Image: e.Image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--invoker-kind=" + e.Invoker.GetTypeMeta().Kind,
			"--invoker-name=" + e.Invoker.GetObjectMeta().Name,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-namespace=" + targetInfo.Target.Ref.Namespace,
			"--restore-model=job",
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", e.Repository.Spec.Backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + metrics.GetPushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
		}, flags.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: apis.KeyNodeName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
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

	// mount tmp volume
	container.VolumeMounts = util.UpsertTmpVolumeMount(container.VolumeMounts)

	// mount the volumes specified in RestoreSession into the job
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		container.VolumeMounts = append(container.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// Pass container runtimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *targetInfo.RuntimeSettings.Container)
	}

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user 65535. We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	securityContext := &core.SecurityContext{
		RunAsUser:  pointer.Int64P(0),
		RunAsGroup: pointer.Int64P(0),
	}
	if targetInfo.RuntimeSettings.Container != nil {
		container.SecurityContext = util.UpsertSecurityContext(securityContext, targetInfo.RuntimeSettings.Container.SecurityContext)
	} else {
		container.SecurityContext = securityContext
	}

	podSpec := core.PodSpec{
		Containers:    []core.Container{container},
		RestartPolicy: core.RestartPolicyNever,
	}

	// Pass pod runtimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Pod != nil {
		podSpec = ofst_util.ApplyPodRuntimeSettings(podSpec, *targetInfo.RuntimeSettings.Pod)
	}

	// add an emptyDir volume for holding temporary files
	podSpec.Volumes = util.UpsertTmpVolume(podSpec.Volumes, targetInfo.TempDir)

	return podSpec
}

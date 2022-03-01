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

package framework

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	shell "github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	meta_util "kmodules.xyz/client-go/meta"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
)

var sampleFiles = []string{"test-data1.txt", "test-data2.txt", "test-data3.txt", "test-data4.txt", "test-data5.txt"}

const (
	PullInterval             = time.Second * 2
	WaitTimeOut              = time.Minute * 10
	TaskPVCBackup            = "pvc-backup"
	TaskPVCRestore           = "pvc-restore"
	TestSourceDataTargetPath = "/source/data"
	TestSourceVolumeAndMount = SourceVolume + ":" + TestSourceDataMountPath
	WrongBackupBlueprintName = "backup-blueprint"
	WrongTargetPath          = "/source/data-1"

	SourceDeployment   = "source-dp"
	RestoredDeployment = "restored-dp"

	SourceStatefulSet   = "source-ss"
	RestoredStatefulSet = "restored-ss"

	SourceDaemonSet   = "source-dmn"
	RestoredDaemonSet = "restored-dmn"

	SourceReplicaSet   = "source-rs"
	RestoredReplicaSet = "restored-rs"

	SourceReplicationController   = "source-rc"
	RestoredReplicationController = "restored-rc"

	SourcePVC = "source-pvc"

	SourceVolume   = "source-volume"
	RestoredVolume = "restored-volume"

	WorkloadBackupBlueprint = "workload-backup-blueprint"
	PvcBackupBlueprint      = "pvc-backup-blueprint"

	TestFSGroup         = 2000
	TestResourceRequest = "200Mi"
	TestResourceLimit   = "300Mi"
	TestUserID          = 2000
)

func (f *Framework) EventuallyEvent(meta metav1.ObjectMeta, involvedObjectKind string) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      involvedObjectKind,
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type":                     core.EventTypeWarning,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	}, WaitTimeOut, PullInterval)
}

func deleteInForeground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func (f *Framework) ReadSampleDataFromFromWorkload(meta metav1.ObjectMeta, resourceKind string) ([]string, error) {
	switch resourceKind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindPod:
		pod, err := f.GetPod(meta)
		if err != nil {
			return nil, err
		}
		set := sets.NewString()
		data, err := f.ExecOnPod(pod, "ls", TestSourceDataMountPath)
		if err != nil {
			return nil, err
		}
		files := strings.Fields(data)
		for i := range files {
			set.Insert(strings.TrimSpace(files[i]))
		}
		return set.List(), nil
	case apis.KindStatefulSet, apis.KindDaemonSet:
		set := sets.NewString()
		pods, err := f.GetAllPods(meta)
		if err != nil {
			return set.List(), err
		}
		for _, pod := range pods {
			data, err := f.ExecOnPod(&pod, "ls", TestSourceDataMountPath)
			if err != nil {
				return set.List(), err
			}
			files := strings.Fields(data)
			for i := range files {
				set.Insert(strings.TrimSpace(files[i]))
			}
		}
		return set.List(), err
	}
	return nil, nil
}

func WaitUntilRepositoryDeleted(sc cs.Interface, repository *api.Repository) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1alpha1().Repositories(repository.Namespace).Get(context.TODO(), repository.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func (f *Framework) GetNodeName(meta metav1.ObjectMeta) string {
	pod, err := f.GetPod(meta)
	if err == nil {
		return pod.Spec.NodeName
	}

	nodes, err := f.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, node := range nodes.Items {
			if !strings.HasSuffix(node.Name, "master") { // for concourse test, master node has "master" suffix in the name.
				return node.Name
			}
		}
	}

	// if none of above succeed, return default testing node "minikube"
	return "minikube"
}

func (f *Framework) CreateSampleDataInsideWorkload(meta metav1.ObjectMeta, resourceKind string) error {
	switch resourceKind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindPod:
		pod, err := f.GetPod(meta)
		if err != nil {
			return err
		}
		_, err = f.ExecOnPod(pod, "touch", filepath.Join(TestSourceDataMountPath, sampleFiles[0]))
		if err != nil {
			return err
		}
	case apis.KindStatefulSet, apis.KindDaemonSet:
		pods, err := f.GetAllPods(meta)
		if err != nil {
			return err
		}
		for i, pod := range pods {
			_, err := f.ExecOnPod(&pod, "touch", filepath.Join(TestSourceDataMountPath, sampleFiles[i]))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (fi *Invocation) CreateSampleFiles(objMeta metav1.ObjectMeta, sampleFiles []string) error {
	pod, err := fi.GetPod(objMeta)
	if err != nil {
		return err
	}
	commands := []string{"touch"}
	for i := range sampleFiles {
		commands = append(commands, filepath.Join(TestSourceDataMountPath, sampleFiles[i]))
	}
	_, err = fi.ExecOnPod(pod, commands...)
	if err != nil {
		return err
	}
	return nil
}

func (fi *Invocation) CleanupSampleDataFromWorkload(meta metav1.ObjectMeta, resourceKind string) error {
	switch resourceKind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindPod:
		pod, err := fi.GetPod(meta)
		if err != nil {
			return err
		}
		_, err = fi.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", TestSourceDataMountPath))
		if err != nil {
			return err
		}
	case apis.KindStatefulSet, apis.KindDaemonSet:
		pods, err := fi.GetAllPods(meta)
		if err != nil {
			return err
		}
		for _, pod := range pods {
			_, err = fi.ExecOnPod(&pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", TestSourceDataMountPath))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (fi *Invocation) AppendToCleanupList(resources ...interface{}) {
	for i := range resources {
		fi.testResources = append(fi.testResources, resources[i])
	}
}

func (fi *Invocation) CleanupTestResources() error {
	// delete all test resources
	By("Cleaning Test Resources")
	for i := range fi.testResources {
		err := fi.DeleteResource(fi.testResources[i])
		if err != nil {
			return err
		}
	}

	// wait until resource has been deleted
	for i := range fi.testResources {
		err := fi.WaitUntilResourceDeleted(fi.testResources[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (fi *Invocation) DeleteResource(obj interface{}) error {
	gvr, objMeta, err := getGVRAndObjectMeta(obj)
	if err != nil {
		return err
	}
	deletionPolicy := meta_util.DeleteInBackground()
	// Repository has finalizer wipeOut finalizer.
	// Hence, we should ensure that it has been deleted before deleting the respective secret.
	if gvr.Resource == api.ResourceKindRepository {
		deletionPolicy = meta_util.DeleteInForeground()
	}
	err = fi.dmClient.Resource(gvr).Namespace(objMeta.Namespace).Delete(context.TODO(), objMeta.Name, deletionPolicy)
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (fi *Invocation) WaitUntilResourceDeleted(obj interface{}) error {
	gvr, objMeta, err := getGVRAndObjectMeta(obj)
	if err != nil {
		return err
	}
	return fi.waitUntilResourceDeleted(gvr, objMeta)
}

func (fi *Invocation) waitUntilResourceDeleted(gvr schema.GroupVersionResource, objMeta metav1.ObjectMeta) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := fi.dmClient.Resource(gvr).Namespace(objMeta.Namespace).Get(context.TODO(), objMeta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func getGVRAndObjectMeta(obj interface{}) (schema.GroupVersionResource, metav1.ObjectMeta, error) {
	switch w := obj.(type) {
	case *apps.Deployment:
		w.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(apis.KindDeployment))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralDeployment}, w.ObjectMeta, nil
	case *apps.DaemonSet:
		w.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(apis.KindDaemonSet))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralDaemonSet}, w.ObjectMeta, nil
	case *apps.StatefulSet:
		w.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(apis.KindStatefulSet))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralStatefulSet}, w.ObjectMeta, nil
	case *apps.ReplicaSet:
		w.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(apis.KindReplicaSet))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralReplicaSet}, w.ObjectMeta, nil
	case *core.ReplicationController:
		w.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindReplicationController))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralReplicationController}, w.ObjectMeta, nil
	case *core.Pod:
		w.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindPod))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralPod}, w.ObjectMeta, nil
	case *ocapps.DeploymentConfig:
		w.GetObjectKind().SetGroupVersionKind(ocapps.GroupVersion.WithKind(apis.KindDeploymentConfig))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralDeploymentConfig}, w.ObjectMeta, nil
	case *core.PersistentVolumeClaim:
		w.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindPersistentVolumeClaim))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralPersistentVolumeClaim}, w.ObjectMeta, nil
	case *appCatalog.AppBinding:
		w.GetObjectKind().SetGroupVersionKind(appCatalog.SchemeGroupVersion.WithKind(apis.KindAppBinding))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralAppBinding}, w.ObjectMeta, nil
	case *v1beta1.BackupConfiguration:
		w.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupConfiguration))
		gvk := w.TypeMeta.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: v1beta1.ResourcePluralBackupConfiguration}, w.ObjectMeta, nil
	case *v1beta1.BackupSession:
		w.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupSession))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: v1beta1.ResourcePluralBackupSession}, w.ObjectMeta, nil
	case *v1beta1.RestoreSession:
		rs := obj.(*v1beta1.RestoreSession)
		rs.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindRestoreSession))
		gvk := rs.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: v1beta1.ResourcePluralRestoreSession}, rs.ObjectMeta, nil
	case *v1beta1.BackupBlueprint:
		w.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupBlueprint))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: v1beta1.ResourcePluralBackupBlueprint}, w.ObjectMeta, nil
	case *api.Repository:
		w.GetObjectKind().SetGroupVersionKind(api.SchemeGroupVersion.WithKind(api.ResourceKindRepository))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: api.ResourcePluralRepository}, w.ObjectMeta, nil
	case *core.Secret:
		w.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindSecret))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralSecret}, w.ObjectMeta, nil
	case *core.Service:
		w.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindService))
		gvk := w.GroupVersionKind()
		return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apis.ResourcePluralService}, w.ObjectMeta, nil
	default:
		return schema.GroupVersionResource{}, metav1.ObjectMeta{}, fmt.Errorf("failed to get GroupVersionResource. Reason: Unknown resource type")
	}
}

func (fi *Invocation) PrintDebugHelpers() {
	const kubectl = "/usr/bin/kubectl"
	sh := shell.NewSession()

	fmt.Println("\n======================================[ Describe BackupSession ]===================================================")
	if err := sh.Command(kubectl, "describe", "backupsession", "-n", fi.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe BackupConfiguration ]==========================================")
	if err := sh.Command(kubectl, "describe", "backupconfiguration", "-n", fi.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe RestoreSession ]==========================================")
	if err := sh.Command(kubectl, "describe", "restoresession", "-n", fi.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe Job ]===================================================")
	if err := sh.Command(kubectl, "describe", "job", "-n", fi.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n===============[ Debug info for Stash sidecar/init-container/backup job/restore job ]===================")
	pods, err := fi.KubeClient.CoreV1().Pods(fi.Namespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	} else {
		for _, pod := range pods.Items {
			debugTarget, containerArgs := isDebugTarget(append(pod.Spec.InitContainers, pod.Spec.Containers...))
			if debugTarget {
				fmt.Printf("\n--------------- Describe Pod: %s -------------------\n", pod.Name)
				if err := sh.Command(kubectl, "describe", "po", "-n", fi.Namespace(), pod.Name).Run(); err != nil {
					fmt.Println(err)
				}

				fmt.Printf("\n---------------- Log from Pod: %s ------------------\n", pod.Name)
				logArgs := []interface{}{"logs", "-n", fi.namespace, pod.Name}
				for i := range containerArgs {
					logArgs = append(logArgs, containerArgs[i])
				}
				err = sh.Command(kubectl, logArgs...).
					Command("cut", "-f", "4-", "-d ").
					Command("awk", `{$2=$2;print}`).
					Command("uniq").Run()
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}

func (f *Framework) PrintOperatorLog() {
	sh := shell.NewSession()
	sh.PipeStdErrors = true

	fmt.Println("\n======================================[ Log from Stash operator ]===================================================")
	pod, err := f.GetOperatorPod()
	if err != nil {
		fmt.Println(err)
	} else {
		err := sh.Command("/usr/bin/kubectl", "logs", "-n", "kube-system", pod.Name, "-c", "operator").
			Command("grep", "-i", "error").
			Command("cut", "-f", "4-", "-d ").
			Command("awk", `{$2=$2;print}`).
			Command("uniq").Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func isDebugTarget(containers []core.Container) (bool, []string) {
	for _, c := range containers {
		if c.Name == "stash" || c.Name == "stash-init" {
			return true, []string{"-c", c.Name}
		} else if strings.HasPrefix(c.Name, "update-status") {
			return true, []string{"--all-containers"}
		}
	}
	return false, nil
}

func (fi *Invocation) HookFailed(involvedObjectKind string, involvedObjectMeta metav1.ObjectMeta, probeType string) (bool, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{
		"involvedObject.kind":      involvedObjectKind,
		"involvedObject.name":      involvedObjectMeta.Name,
		"involvedObject.namespace": involvedObjectMeta.Namespace,
		"type":                     core.EventTypeWarning,
	})
	events, err := fi.KubeClient.CoreV1().Events(fi.namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: fieldSelector.String()})
	Expect(err).NotTo(HaveOccurred())

	hasHookFailureEvent := false
	for _, e := range events.Items {
		if strings.Contains(e.Message, fmt.Sprintf("failed to execute %q probe.", probeType)) {
			hasHookFailureEvent = true
			break
		}
	}
	return hasHookFailureEvent, nil
}

func (f *Framework) EventuallyEventWritten(involvedObjectMeta metav1.ObjectMeta, involvedObjectKind, eventType, eventReason string) GomegaAsyncAssertion {
	return Eventually(func() bool {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      involvedObjectKind,
			"involvedObject.name":      involvedObjectMeta.Name,
			"involvedObject.namespace": involvedObjectMeta.Namespace,
			"type":                     eventType,
		})
		events, err := f.KubeClient.CoreV1().Events(involvedObjectMeta.Namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: fieldSelector.String()})
		if err != nil {
			return false
		}
		for _, event := range events.Items {
			if event.Reason == eventReason {
				return true
			}
		}
		return false
	}, WaitTimeOut, PullInterval)
}

func HasFSGroup(sc *core.PodSecurityContext) bool {
	return sc != nil && sc.FSGroup != nil && *sc.FSGroup == TestFSGroup
}

func HasResources(containers []core.Container) bool {
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "stash") ||
			strings.HasPrefix(c.Name, "pvc-") ||
			strings.HasPrefix(c.Name, "update-status") {
			if c.Resources.Limits.Memory() != nil &&
				c.Resources.Requests.Memory() != nil &&
				c.Resources.Limits.Memory().String() == TestResourceLimit &&
				c.Resources.Requests.Memory().String() == TestResourceRequest {
				return true
			}
		}
	}
	return false
}

func HasSecurityContext(containers []core.Container) bool {
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "stash") ||
			strings.HasPrefix(c.Name, "pvc-") ||
			strings.HasPrefix(c.Name, "update-status") {
			if c.SecurityContext != nil &&
				c.SecurityContext.RunAsUser != nil &&
				*c.SecurityContext.RunAsUser == TestUserID {
				return true
			}
		}
	}
	return false
}

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

package framework

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"stash.appscode.dev/stash/apis"
	rep "stash.appscode.dev/stash/apis/repositories/v1alpha1"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"

	"github.com/appscode/go/sets"
	shell "github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"kmodules.xyz/client-go/dynamic"
	"kmodules.xyz/client-go/meta"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
)

var (
	files = []string{"test-data1.txt", "test-data2.txt", "test-data3.txt", "test-data4.txt", "test-data5.txt"}
)

const (
	TestSoucreDemoDataPath    = "/data/stash-test/demo-data"
	TestSourceDataDir1        = "/source/data/dir-1"
	TestSourceDataDir2        = "/source/data/dir-2"
	KindRestic                = "Restic"
	KindRepository            = "Repository"
	KindRecovery              = "Recovery"
	PullInterval              = time.Second * 2
	WaitTimeOut               = time.Minute * 3
	TaskPVCBackup             = "pvc-backup"
	TaskPVCRestore            = "pvc-restore"
	TestSourceDataTargetPath  = "/source/data"
	TestSourceDataVolumeMount = "source-data:/source/data"
	WrongBackupBlueprintName  = "backup-blueprint"
	WrongTargetPath           = "/source/data-1"

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

	SourceVolume   = "source-volume"
	RestoredVolume = "restored-volume"

	WorkloadBackupBlueprint = "workload-backup-blueprint"
	PvcBackupBlueprint      = "pvc-backup-blueprint"
)

func (f *Framework) EventualEvent(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Repository",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type":                     core.EventTypeNormal,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) EventualWarning(meta metav1.ObjectMeta, involvedObjectKind string) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      involvedObjectKind,
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type":                     core.EventTypeWarning,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) EventuallyEvent(meta metav1.ObjectMeta, involvedObjectKind string) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      involvedObjectKind,
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type":                     core.EventTypeWarning,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) CountSuccessfulBackups(events []core.Event) int {
	count := 0
	for _, e := range events {
		if e.Reason == eventer.EventReasonSuccessfulBackup {
			count++
		}
	}
	return count
}

func (f *Framework) CountFailedSetup(events []core.Event) int {
	count := 0
	for _, e := range events {
		if e.Reason == eventer.EventReasonFailedSetup {
			count++
		}
	}
	return count
}

func deleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func deleteInForeground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func CleanupMinikubeHostPath() error {
	cmd := "minikube"
	args := []string{"ssh", "sudo rm -rf /data/stash-test"}
	return exec.Command(cmd, args...).Run()
}

func (f *Framework) DeleteJobAndDependents(jobName string, recovery *api.Recovery) {
	By("Checking Job deleted")
	Eventually(func() bool {
		_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get(jobName, metav1.GetOptions{})
		return kerr.IsNotFound(err) || kerr.IsGone(err)
	}, time.Minute*3, time.Second*2).Should(BeTrue())

	By("Checking pods deleted")
	Eventually(func() bool {
		pods, err := f.KubeClient.CoreV1().Pods(recovery.Namespace).List(metav1.ListOptions{
			LabelSelector: "job-name=" + jobName, // pods created by job has a job-name label
		})
		Expect(err).NotTo(HaveOccurred())
		return len(pods.Items) == 0
	}, time.Minute*3, time.Second*2).Should(BeTrue())

	By("Checking service-account deleted")
	Eventually(func() bool {
		_, err := f.KubeClient.CoreV1().ServiceAccounts(recovery.Namespace).Get(jobName, metav1.GetOptions{})
		return kerr.IsNotFound(err) || kerr.IsGone(err)
	}, time.Minute*3, time.Second*2).Should(BeTrue())

	By("Checking role-binding deleted")
	Eventually(func() bool {
		_, err := f.KubeClient.RbacV1().RoleBindings(recovery.Namespace).Get(jobName, metav1.GetOptions{})
		return kerr.IsNotFound(err) || kerr.IsGone(err)
	}, time.Minute*3, time.Second*2).Should(BeTrue())

	By("Checking repo-reader role-binding deleted")
	Eventually(func() bool {
		_, err := f.KubeClient.RbacV1().RoleBindings(recovery.Spec.Repository.Namespace).Get(stash_rbac.GetRepoReaderRoleBindingName(jobName, recovery.Namespace), metav1.GetOptions{})
		return kerr.IsNotFound(err) || kerr.IsGone(err)
	}, time.Minute*3, time.Second*2).Should(BeTrue())
}

func (f *Framework) CreateDemoData(meta metav1.ObjectMeta) error {
	err := f.CreateDirectory(meta, []string{TestSourceDataDir1, TestSourceDataDir2})
	if err != nil {
		return err
	}
	err = f.CreateDataOnMountedDir(meta, []string{TestSourceDataDir1}, "file1.txt")
	if err != nil {
		return err
	}
	err = f.CreateDataOnMountedDir(meta, []string{TestSourceDataDir2}, "file2.txt")
	if err != nil {
		return err
	}
	return nil
}

func (f *Framework) ReadSampleDataFromMountedDirectory(meta metav1.ObjectMeta, paths []string, resourceKind string) ([]string, error) {
	switch resourceKind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController:
		pod, err := f.GetPod(meta)
		if err != nil {
			return nil, err
		}
		var data string
		datas := make([]string, 0)
		for _, p := range paths {
			data, err = f.ExecOnPod(pod, "ls", p)
			if err != nil {
				return nil, err
			}
			datas = append(datas, data)
		}
		return datas, err
	case apis.KindStatefulSet, apis.KindDaemonSet:
		datas := make([]string, 0)
		pods, err := f.GetAllPods(meta)
		if err != nil {
			return datas, err
		}
		for _, path := range paths {
			for _, pod := range pods {
				data, err := f.ExecOnPod(&pod, "ls", path)
				if err != nil {
					continue
				}
				datas = append(datas, data)
			}
		}
		return datas, err
	}
	return []string{}, nil
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
		set.Insert(strings.TrimSpace(data))
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
			set.Insert(strings.TrimSpace(data))
		}
		return set.List(), err
	}
	return nil, nil
}

func (f *Framework) CreateDirectory(meta metav1.ObjectMeta, directories []string) error {
	pod, err := f.GetPod(meta)
	if err != nil {
		return err
	}

	for _, dir := range directories {
		_, err := f.ExecOnPod(pod, "mkdir", "-p", dir)
		if err != nil {
			return err
		}
	}
	return nil
}
func (f *Framework) CreateDataOnMountedDir(meta metav1.ObjectMeta, paths []string, fileName string) error {
	pod, err := f.GetPod(meta)
	if err != nil {
		return err
	}
	for _, path := range paths {
		_, err := f.ExecOnPod(pod, "touch", filepath.Join(path, fileName))
		if err != nil {
			return err
		}

	}
	return nil
}

func (f *Invocation) HostPathVolumeWithMultipleDirectory() []core.Volume {
	return []core.Volume{
		{
			Name: TestSourceDataVolumeName,
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: filepath.Join(TestSoucreDemoDataPath, f.app),
				},
			},
		},
	}
}

func FileGroupsForHostPathVolumeWithMultipleDirectory() []api.FileGroup {
	return []api.FileGroup{
		{
			Path:                TestSourceDataDir1,
			RetentionPolicyName: "keep-last-5",
		},
		{
			Path:                TestSourceDataDir2,
			RetentionPolicyName: "keep-last-5",
		},
	}
}

func (f *Invocation) RecoveredVolume() []core.Volume {
	return []core.Volume{
		{
			Name: TestSourceDataVolumeName,
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: filepath.Join(TestRecoveredVolumePath, f.App()),
				},
			},
		},
	}
}
func (f *Invocation) CleanupRecoveredVolume(meta metav1.ObjectMeta) error {
	pod, err := f.GetPod(meta)
	if err != nil {
		return err
	}
	_, err = f.ExecOnPod(pod, "rm", "-rf", TestSourceDataMountPath)
	if err != nil {
		return err
	}
	return nil
}

func (f *Framework) DaemonSetRepos(daemon *apps.DaemonSet) []*api.Repository {
	return f.GetRepositories(KindMetaReplicas{Kind: apis.KindDaemonSet, Meta: daemon.ObjectMeta, Replicas: 1})
}

func (f *Framework) DeploymentRepos(deployment *apps.Deployment) []*api.Repository {
	return f.GetRepositories(KindMetaReplicas{Kind: apis.KindDeployment, Meta: deployment.ObjectMeta, Replicas: int(*deployment.Spec.Replicas)})
}

func (f *Framework) ReplicationControllerRepos(rc *core.ReplicationController) []*api.Repository {
	return f.GetRepositories(KindMetaReplicas{Kind: apis.KindReplicationController, Meta: rc.ObjectMeta, Replicas: int(*rc.Spec.Replicas)})
}

func (f *Framework) ReplicaSetRepos(rs *apps.ReplicaSet) []*api.Repository {
	return f.GetRepositories(KindMetaReplicas{Kind: apis.KindReplicaSet, Meta: rs.ObjectMeta, Replicas: int(*rs.Spec.Replicas)})
}

func (f *Framework) StatefulSetRepos(ss *apps.StatefulSet) []*api.Repository {
	return f.GetRepositories(KindMetaReplicas{Kind: apis.KindStatefulSet, Meta: ss.ObjectMeta, Replicas: int(*ss.Spec.Replicas)})
}

func (f *Framework) LatestSnapshot(snapshots []rep.Snapshot) rep.Snapshot {
	latestSnapshot := snapshots[0]
	for _, snap := range snapshots {
		if snap.CreationTimestamp.After(latestSnapshot.CreationTimestamp.Time) {
			latestSnapshot = snap
		}
	}
	return latestSnapshot
}

func GetPathsFromResticFileGroups(restic *api.Restic) []string {
	paths := make([]string, 0)
	for _, fg := range restic.Spec.FileGroups {
		paths = append(paths, fg.Path)
	}
	return paths
}

func GetPathsFromRestoreSession(restoreSession *v1beta1.RestoreSession) []string {
	paths := make([]string, 0)
	for i := range restoreSession.Spec.Rules {
		paths = append(paths, restoreSession.Spec.Rules[i].Paths...)
	}
	paths = removeDuplicates(paths)
	return paths
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

func (f *Framework) EventuallyJobSucceed(name string) GomegaAsyncAssertion {
	jobCreated := false
	return Eventually(func() bool {
		obj, err := f.KubeClient.BatchV1().Jobs(f.namespace).Get(name, metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		if err != nil && kerr.IsNotFound(err) && jobCreated {
			return true
		}

		jobCreated = true
		return obj.Status.Succeeded > 0
	}, time.Minute*5, time.Second*5)
}

func WaitUntilNamespaceDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.CoreV1().Namespaces().Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilDeploymentDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.AppsV1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilDaemonSetDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilStatefulSetDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.AppsV1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilReplicaSetDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilReplicationControllerDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilSecretDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilServiceDeleted(kc kubernetes.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := kc.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilResticDeleted(sc cs.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1alpha1().Restics(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilBackupConfigurationDeleted(sc cs.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1beta1().BackupConfigurations(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilRestoreSessionDeleted(sc cs.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1beta1().RestoreSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilRecoveryDeleted(sc cs.Interface, meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1alpha1().Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func WaitUntilRepositoriesDeleted(sc cs.Interface, repositories []*api.Repository) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		allDeleted := true
		for _, repo := range repositories {
			if _, err := sc.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{}); err != nil {
				if kerr.IsNotFound(err) {
					continue
				} else {
					return true, err
				}
			}
			allDeleted = false
		}
		if allDeleted {
			return true, nil
		}
		return false, nil
	})
}
func WaitUntilRepositoryDeleted(sc cs.Interface, repository *api.Repository) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := sc.StashV1alpha1().Repositories(repository.Namespace).Get(repository.Name, metav1.GetOptions{}); err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			} else {
				return true, err
			}
		}
		return false, nil
	})
}

func (f *Framework) WaitUntilDaemonPodReady(meta metav1.ObjectMeta) error {

	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		pod, err := f.GetPod(meta)
		if err != nil {
			return false, nil
		}
		if pod.Status.Phase == core.PodPhase(core.PodRunning) {
			return true, nil
		}
		return false, nil

	})
}

func (f *Framework) GetNodeName(meta metav1.ObjectMeta) string {
	pod, err := f.GetPod(meta)
	if err == nil {
		return pod.Spec.NodeName
	}

	nodes, err := f.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
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
		_, err = f.ExecOnPod(pod, "touch", filepath.Join(TestSourceDataMountPath, files[0]))
		if err != nil {
			return err
		}
	case apis.KindStatefulSet, apis.KindDaemonSet:
		pods, err := f.GetAllPods(meta)
		if err != nil {
			return err
		}
		for i, pod := range pods {
			_, err := f.ExecOnPod(&pod, "touch", filepath.Join(TestSourceDataMountPath, files[i]))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Invocation) CleanupSampleDataFromWorkload(meta metav1.ObjectMeta, resourceKind string) error {

	switch resourceKind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindPod:
		pod, err := f.GetPod(meta)
		if err != nil {
			return err
		}
		_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", TestSourceDataMountPath))
		if err != nil {
			return err
		}
	case apis.KindStatefulSet, apis.KindDaemonSet:
		pods, err := f.GetAllPods(meta)
		if err != nil {
			return err
		}
		for _, pod := range pods {
			_, err = f.ExecOnPod(&pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", TestSourceDataMountPath))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Invocation) ReadDataFromPod(meta metav1.ObjectMeta) (data string, err error) {
	pod, err := f.GetPod(meta)
	if err != nil {
		return "", err
	}
	data, err = f.ExecOnPod(pod, "ls", "-R", TestSourceDataMountPath)
	return data, err
}

func (f *Invocation) AppendToCleanupList(resources ...interface{}) {
	for i := range resources {
		f.testResources = append(f.testResources, resources[i])
	}
}

func (f *Invocation) CleanupTestResources() error {
	// delete all test resources
	By("Cleaning Test Resources")
	for i := range f.testResources {
		gvr, objMeta, err := getGVRAndObjectMeta(f.testResources[i])
		if err != nil {
			return err
		}
		err = f.dmClient.Resource(gvr).Namespace(objMeta.Namespace).Delete(objMeta.Name, deleteInBackground())
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}

	// wait until resource has been deleted
	for i := range f.testResources {
		gvr, objMeta, err := getGVRAndObjectMeta(f.testResources[i])
		if err != nil {
			return err
		}
		err = f.waitUntilResourceDeleted(gvr, objMeta)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Invocation) waitUntilResourceDeleted(gvr schema.GroupVersionResource, objMeta metav1.ObjectMeta) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		if _, err := f.dmClient.Resource(gvr).Namespace(objMeta.Namespace).Get(objMeta.Name, metav1.GetOptions{}); err != nil {
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

func (f *Invocation) AddAnnotations(annotations map[string]string, obj interface{}) error {
	schm := scheme.Scheme
	gvr, _, _ := getGVRAndObjectMeta(obj)
	cur := &unstructured.Unstructured{}
	err := schm.Convert(obj, cur, nil)
	if err != nil {
		return err
	}
	mod := cur.DeepCopy()
	mod.SetAnnotations(annotations)
	out, _, err := dynamic.PatchObject(f.dmClient, gvr, cur, mod)
	if err != nil {

		return err
	}
	err = schm.Convert(out, obj, nil)
	if err != nil {
		return err
	}
	return nil
}

func (f *Framework) EventuallyAnnotationsFound(expectedAnnotations map[string]string, obj interface{}) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			schm := scheme.Scheme
			cur := &unstructured.Unstructured{}
			err := schm.Convert(obj, cur, nil)
			if err != nil {
				return false
			}
			annotations := cur.GetAnnotations()
			for k, v := range expectedAnnotations {
				if !(meta.HasKey(annotations, k) && annotations[k] == v) {
					return false
				}
			}
			return true
		},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Invocation) PrintDebugHelpers() {
	const kubectl = "/usr/bin/kubectl"
	sh := shell.NewSession()

	fmt.Println("\n======================================[ Describe BackupSession ]===================================================")
	if err := sh.Command(kubectl, "describe", "backupsession", "-n", f.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe BackupConfiguration ]==========================================")
	if err := sh.Command(kubectl, "describe", "backupconfiguration", "-n", f.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe RestoreSession ]==========================================")
	if err := sh.Command(kubectl, "describe", "restoresession", "-n", f.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n======================================[ Describe Job ]===================================================")
	if err := sh.Command(kubectl, "describe", "job", "-n", f.Namespace()).Run(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("\n===============[ Debug info for Stash sidecar/init-container/backup job/restore job ]===================")
	pods, err := f.KubeClient.CoreV1().Pods(f.Namespace()).List(metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	} else {
		for _, pod := range pods.Items {
			debugTarget, containerArgs := isDebugTarget(append(pod.Spec.InitContainers, pod.Spec.Containers...))
			if debugTarget {
				fmt.Printf("\n--------------- Describe Pod: %s -------------------\n", pod.Name)
				if err := sh.Command(kubectl, "describe", "po", "-n", f.Namespace(), pod.Name).Run(); err != nil {
					fmt.Println(err)
				}

				fmt.Printf("\n---------------- Log from Pod: %s ------------------\n", pod.Name)
				logArgs := []interface{}{"logs", "-n", f.namespace, pod.Name}
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

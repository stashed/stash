package framework

import (
	"os/exec"
	"time"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const (
	TestSoucreDemoDataPath = "/data/stash-test/demo-data"
	TestSourceDataDir1     = "/source/data/dir-1"
	TestSourceDataDir2     = "/source/data/dir-2"
)

func (f *Framework) EventualEvent(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Repository",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type": core.EventTypeNormal,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) EventualWarning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Repository",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type": core.EventTypeWarning,
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
}

func CreateDemoDataInHostPath() error {
	cmd := "minikube"

	//create directories
	args := []string{"ssh", "sudo mkdir -p /data/stash-test/demo-data/{dir-1,dir-2}"}
	err := exec.Command(cmd, args...).Run()
	if err != nil {
		return err
	}

	//create files in the directories
	args = []string{"ssh", "sudo touch /data/stash-test/demo-data/{dir-1/file1.txt,dir-2/file2.txt}"}
	err = exec.Command(cmd, args...).Run()
	if err != nil {
		return err
	}
	return nil
}

func HostPathVolumeWithMultipleDirectory() []core.Volume {
	return []core.Volume{
		{
			Name: TestSourceDataVolumeName,
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: TestSoucreDemoDataPath,
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

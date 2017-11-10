package framework

import (
	"os/exec"
	"time"

	"github.com/appscode/stash/pkg/eventer"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const (
	updateRetryInterval = 10 * 1000 * 1000 * time.Nanosecond
	maxAttempts         = 5
)

func (f *Framework) EventualEvent(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []core.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Restic",
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
			"involvedObject.kind":      "Restic",
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

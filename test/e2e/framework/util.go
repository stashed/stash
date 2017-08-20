package framework

import (
	"time"

	"github.com/appscode/stash/pkg/eventer"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	updateRetryInterval = 10 * 1000 * 1000 * time.Nanosecond
	maxAttempts         = 5
)

func (f *Framework) EventualEvent(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []apiv1.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Restic",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type": apiv1.EventTypeNormal,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) EventualWarning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []apiv1.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Restic",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type": apiv1.EventTypeWarning,
		})
		events, err := f.KubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) CountSuccessfulBackups(events []apiv1.Event) int {
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

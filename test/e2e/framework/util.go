package framework

import (
	"errors"
	"time"

	"github.com/appscode/log"
	"github.com/appscode/stash/pkg/eventer"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) EventualEvent(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() []apiv1.Event {
		fieldSelector := fields.SelectorFromSet(fields.Set{
			"involvedObject.kind":      "Restic",
			"involvedObject.name":      meta.Name,
			"involvedObject.namespace": meta.Namespace,
			"type": apiv1.EventTypeNormal,
		})
		events, err := f.kubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
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
		events, err := f.kubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		Expect(err).NotTo(HaveOccurred())
		return events.Items
	})
}

func (f *Framework) WaitForBackupEvent(objName string) error {
	try := 0
	fieldSelector := fields.SelectorFromSet(fields.Set{
		"involvedObject.kind":      "Stash",
		"involvedObject.name":      objName,
		"involvedObject.namespace": f.namespace,
		"type": apiv1.EventTypeNormal,
	})
	for {
		events, err := f.kubeClient.CoreV1().Events(f.namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		if err == nil {
			for _, e := range events.Items {
				if e.Reason == eventer.EventReasonSuccessfulBackup {
					return nil
				}
			}
		}
		if try > 12 {
			return err
		}
		log.Infoln("Waiting for 10 second for events of backup process")
		time.Sleep(time.Second * 3)
		try++
	}
	return errors.New("Stash backup failed.")
}

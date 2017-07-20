package framework

import (
	"fmt"
	"time"

	"github.com/appscode/log"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_HEADLESS_SERVICE = "headless"
)

func (fi *Invocation) HeadlessService() apiv1.Service {
	return apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TEST_HEADLESS_SERVICE,
			Namespace: fi.namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": fi.app,
			},
			ClusterIP: apiv1.ClusterIPNone,
			Ports: []apiv1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func (f *Framework) CreateService(obj apiv1.Service) error {
	_, err := f.kubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) UpdateService(meta metav1.ObjectMeta, transformer func(apiv1.Service) apiv1.Service) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.CoreV1().Services(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update Service %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update Service %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) DeleteService(meta metav1.ObjectMeta) error {
	return f.kubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

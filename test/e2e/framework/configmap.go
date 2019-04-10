package framework

import (
	"encoding/json"
	"fmt"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) CheckLeaderElection(meta metav1.ObjectMeta, kind string, modifier string) {
	var podName string

	By("Waiting for configmap annotation")
	Eventually(func() bool {
		var err error
		if podName, err = f.GetLeaderIdentity(meta, kind, modifier); err != nil {
			return false
		}
		return true
	}).Should(BeTrue())

	By("Deleting leader pod: " + podName)
	err := f.KubeClient.CoreV1().Pods(meta.Namespace).Delete(podName, &metav1.DeleteOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	By("Waiting for reconfigure configmap annotation")
	Eventually(func() bool {
		if podNameNew, err := f.GetLeaderIdentity(meta, kind, modifier); err != nil || podNameNew == podName {
			return false
		}
		return true
	}).Should(BeTrue())
}

func (f *Framework) GetLeaderIdentity(meta metav1.ObjectMeta, kind string, modifier string) (string, error) {
	var configMapLockName string
	switch modifier {
	case api.ResourceKindRestic:
		configMapLockName = util.GetConfigmapLockName(api.LocalTypedReference{
			Kind: kind,
			Name: meta.Name,
		})
	case v1beta1.ResourceKindBackupConfiguration:
		configMapLockName = util.GetBackupConfigmapLockName(v1beta1.TargetRef{
			Kind: kind,
			Name: meta.Name,
		})
	case v1beta1.ResourceKindRestoreSession:
		configMapLockName = util.GetRestoreConfigmapLockName(v1beta1.TargetRef{
			Kind: kind,
			Name: meta.Name,
		})
	}
	annotationKey := "control-plane.alpha.kubernetes.io/leader"
	idKey := "holderIdentity"

	configMap, err := f.KubeClient.CoreV1().ConfigMaps(meta.Namespace).Get(configMapLockName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	annotationValue, ok := configMap.Annotations[annotationKey]
	if !ok || annotationValue == "" {
		return "", fmt.Errorf("key not found: %s", annotationKey)
	}
	valueMap := make(map[string]interface{})
	if err = json.Unmarshal([]byte(annotationValue), &valueMap); err != nil {
		return "", err
	}
	id, ok := valueMap[idKey]
	if !ok || id == "" {
		return "", fmt.Errorf("key not found: %s", idKey)
	}
	return id.(string), nil
}

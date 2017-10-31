package framework

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/stash/pkg/scheduler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) GetLeaderAnnotation(meta metav1.ObjectMeta) (string, error) {
	configMapName := scheduler.ConfigMapPrefix + meta.Name
	annotationKey := "control-plane.alpha.kubernetes.io/leader"
	idKey := "holderIdentity"

	configMap, err := f.KubeClient.CoreV1().ConfigMaps(meta.Namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	annotationValue, ok := configMap.Annotations[annotationKey]
	if !ok {
		return "", fmt.Errorf("key not found: %s", annotationKey)
	}
	valueMap := make(map[string]interface{})
	if err = json.Unmarshal([]byte(annotationValue), &valueMap); err != nil {
		return "", err
	}
	id, ok := valueMap[idKey]
	if !ok {
		return "", fmt.Errorf("key not found: %s", idKey)
	}
	return id.(string), nil
}

package v1beta1

import (
	"github.com/json-iterator/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var json = jsoniter.ConfigFastest

func IsOwnedByDeployment(references []metav1.OwnerReference) bool {
	for _, ref := range references {
		if ref.Kind == "Deployment" && ref.Name != "" {
			return true
		}
	}
	return false
}

package api

import (
	"encoding/json"
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/runtime"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	// Add field label conversions for kinds having selectable nothing but ObjectMeta fields.
	var err error
	for _, k := range []string{"Ingress", "Alert", "Backup", "Certificate"} {
		kind := k // don't close over range variables
		err = api.Scheme.AddFieldLabelConversionFunc("appscode.com/v1", kind,
			func(label, value string) (string, string, error) {
				switch label {
				case "metadata.name", "metadata.namespace":
					return label, value, nil
				default:
					return "", "", fmt.Errorf("field label %q not supported for %q", label, kind)
				}
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewEngressFromIngress(ing interface{}) (*Ingress, error) {
	data, err := json.Marshal(ing)
	if err != nil {
		return nil, err
	}
	engress := &Ingress{}
	err = json.Unmarshal(data, engress)
	if err != nil {
		return nil, err
	}
	if engress.Annotations == nil {
		engress.Annotations = make(map[string]string)
	}
	engress.Annotations[EngressKind] = "ingress"
	return engress, nil
}

func NewIngressFromEngress(ing interface{}) (*extensions.Ingress, error) {
	data, err := json.Marshal(ing)
	if err != nil {
		return nil, err
	}
	engress := &extensions.Ingress{}
	err = json.Unmarshal(data, engress)
	if err != nil {
		return nil, err
	}
	if engress.Annotations == nil {
		engress.Annotations = make(map[string]string)
	}
	engress.Annotations[EngressKind] = EngressKey + "/" + V1beta1SchemeGroupVersion.Version
	return engress, nil
}

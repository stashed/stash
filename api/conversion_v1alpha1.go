package api

import (
	"fmt"
apiv1 "k8s.io/client-go/pkg/api/v1"
"k8s.io/apimachinery/pkg/runtime"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	// Add field label conversions for kinds having selectable nothing but ObjectMeta fields.
	var err error
	for _, k := range []string{"Restik"} {
		kind := k // don't close over range variables
		err = apiv1.Scheme.AddFieldLabelConversionFunc("backup.appscode.com/v1", kind,
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

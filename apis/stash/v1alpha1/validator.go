package v1alpha1

import (
	"fmt"

	"gopkg.in/robfig/cron.v2"
)

func (r Restic) IsValid() error {
	_, err := cron.Parse(r.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("spec.schedule %s is invalid. Reason: %s", r.Spec.Schedule, err)
	}
	if r.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("missing repository secret name")
	}
	return nil
}

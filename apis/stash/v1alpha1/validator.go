package v1alpha1

import (
	"fmt"
	"strings"

	cron "github.com/robfig/cron/v3"
)

func (r Restic) IsValid() error {
	for i, fg := range r.Spec.FileGroups {
		if fg.RetentionPolicyName == "" {
			continue
		}

		found := false
		for _, policy := range r.Spec.RetentionPolicies {
			if policy.Name == fg.RetentionPolicyName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("spec.fileGroups[%d].retentionPolicyName %s is not found", i, fg.RetentionPolicyName)
		}
	}

	_, err := cron.ParseStandard(r.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("spec.schedule %s is invalid. Reason: %s", r.Spec.Schedule, err)
	}
	if r.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("missing repository secret name")
	}
	return nil
}

func (r Recovery) IsValid() error {
	if len(r.Spec.Paths) == 0 {
		return fmt.Errorf("missing filegroup paths")
	}
	if len(r.Spec.RecoveredVolumes) == 0 {
		return fmt.Errorf("missing recovery volume")
	}

	if r.Spec.Repository.Name == "" {
		return fmt.Errorf("missing repository name")
	} else {
		if !(strings.HasPrefix(r.Spec.Repository.Name, "deployment.") ||
			strings.HasPrefix(r.Spec.Repository.Name, "replicationcontroller.") ||
			strings.HasPrefix(r.Spec.Repository.Name, "replicaset.") ||
			strings.HasPrefix(r.Spec.Repository.Name, "statefulset.") ||
			strings.HasPrefix(r.Spec.Repository.Name, "daemonset.")) {
			return fmt.Errorf("invalid repository name")
		}
	}
	if r.Spec.Repository.Namespace == "" {
		return fmt.Errorf("missing repository namespace")
	}
	if r.Spec.Snapshot != "" {
		if !strings.HasPrefix(r.Spec.Snapshot, r.Spec.Repository.Name+"-") {
			return fmt.Errorf("invalid snapshot name")
		}
	}
	return nil
}

func (r Repository) IsValid() error {
	if r.Spec.WipeOut {
		if r.Spec.Backend.Local != nil {
			return fmt.Errorf("wipe out operation is not supported for local backend")
		} else if r.Spec.Backend.B2 != nil {
			return fmt.Errorf("wipe out operation is not supported for B2 backend")
		}
	}
	return nil
}

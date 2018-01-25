package v1alpha1

import (
	"fmt"

	"gopkg.in/robfig/cron.v2"
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

	_, err := cron.Parse(r.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("spec.schedule %s is invalid. Reason: %s", r.Spec.Schedule, err)
	}
	if r.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("missing repository secret name")
	}
	return nil
}

func (r Recovery) IsValid() error {
	if r.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("missing repository secret name")
	}
	if len(r.Spec.Paths) == 0 {
		return fmt.Errorf("missing filegroup paths")
	}
	if len(r.Spec.RecoveredVolumes) == 0 {
		return fmt.Errorf("missing recovery volume")
	}

	if err := r.Spec.Workload.Canonicalize(); err != nil {
		return err
	}

	switch r.Spec.Workload.Kind {
	case KindDeployment, KindReplicaSet, KindReplicationController:
		if r.Spec.PodOrdinal != "" || r.Spec.NodeName != "" {
			return fmt.Errorf("should not specify podOrdinal/nodeSelector for workload kind %s", r.Spec.Workload.Kind)
		}
	case KindStatefulSet:
		if r.Spec.PodOrdinal == "" {
			return fmt.Errorf("must specify podOrdinal for workload kind %s", r.Spec.Workload.Kind)
		}
		if r.Spec.NodeName != "" {
			return fmt.Errorf("should not specify nodeSelector for workload kind %s", r.Spec.Workload.Kind)
		}
	case KindDaemonSet:
		if r.Spec.NodeName == "" {
			return fmt.Errorf("must specify nodeSelector for workload kind %s", r.Spec.Workload.Kind)
		}
		if r.Spec.PodOrdinal != "" {
			return fmt.Errorf("should not specify podOrdinal for workload kind %s", r.Spec.Workload.Kind)
		}
	}
	return nil
}

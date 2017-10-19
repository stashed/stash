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

func (r Recovery) IsValid() error {
	if r.Spec.Restic == "" {
		return fmt.Errorf("missing restic name")
	}
	if len(r.Spec.Volumes) == 0 {
		return fmt.Errorf("missing target vollume")
	}

	if r.Spec.Workload == "" {
		return fmt.Errorf("missing workload")
	}
	appKind, _, err := ExtractWorkload(r.Spec.Workload)
	if err != nil {
		return err
	}
	switch appKind {
	case AppKindDeployment, AppKindReplicaSet, AppKindReplicationController:
		if len(r.Spec.PodOrdinal) != 0 || len(r.Spec.NodeSelector) != 0 {
			return fmt.Errorf("should not specify podOrdinal/nodeSelector for workload kind %s", appKind)
		}
	case AppKindStatefulSet:
		if len(r.Spec.PodOrdinal) == 0 {
			return fmt.Errorf("must specify podOrdinal for workload kind %s", appKind)
		}
		if len(r.Spec.NodeSelector) != 0 {
			return fmt.Errorf("should not specify nodeSelector for workload kind %s", appKind)
		}
	case AppKindDaemonSet:
		if len(r.Spec.NodeSelector) == 0 {
			return fmt.Errorf("must specify nodeSelector for workload kind %s", appKind)
		}
		if len(r.Spec.PodOrdinal) != 0 {
			return fmt.Errorf("should not specify podOrdinal for workload kind %s", appKind)
		}
	}
	return nil
}

package v1alpha1

import (
	"fmt"
	"strings"

	"stash.appscode.dev/stash/apis"
)

// LocalTypedReference contains enough information to let you inspect or modify the referred object.
type LocalTypedReference struct {
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	Kind string `json:"kind,omitempty"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	// +optional
	Name string `json:"name,omitempty"`
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

func (workload *LocalTypedReference) Canonicalize() error {
	if workload.Name == "" || workload.Kind == "" {
		return fmt.Errorf("missing workload name or kind")
	}
	switch strings.ToLower(workload.Kind) {
	case "deployments", "deployment", "deploy":
		workload.Kind = apis.KindDeployment
	case "replicasets", "replicaset", "rs":
		workload.Kind = apis.KindReplicaSet
	case "replicationcontrollers", "replicationcontroller", "rc":
		workload.Kind = apis.KindReplicationController
	case "statefulsets", "statefulset":
		workload.Kind = apis.KindStatefulSet
	case "daemonsets", "daemonset", "ds":
		workload.Kind = apis.KindDaemonSet
	default:
		return fmt.Errorf(`unrecognized workload "Kind" %v`, workload.Kind)
	}
	return nil
}

func (workload LocalTypedReference) GetRepositoryCRDName(podName, nodeName string) string {
	name := ""
	switch workload.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController:
		name = strings.ToLower(workload.Kind) + "." + workload.Name
	case apis.KindStatefulSet:
		name = strings.ToLower(workload.Kind) + "." + podName
	case apis.KindDaemonSet:
		name = strings.ToLower(workload.Kind) + "." + workload.Name + "." + nodeName
	}
	return name
}

func (workload LocalTypedReference) HostnamePrefix(podName, nodeName string) (hostname, prefix string, err error) {
	if err := workload.Canonicalize(); err != nil {
		return "", "", err
	}

	if workload.Name == "" || workload.Kind == "" {
		return "", "", fmt.Errorf("missing workload name or kind")
	}
	switch workload.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController:
		return workload.Name, strings.ToLower(workload.Kind) + "/" + workload.Name, nil
	case apis.KindStatefulSet:
		if podName == "" {
			return "", "", fmt.Errorf("missing podName for %s", apis.KindStatefulSet)
		}
		return podName, strings.ToLower(workload.Kind) + "/" + podName, nil
	case apis.KindDaemonSet:
		if nodeName == "" {
			return "", "", fmt.Errorf("missing nodeName for %s", apis.KindDaemonSet)
		}
		return nodeName, strings.ToLower(workload.Kind) + "/" + workload.Name + "/" + nodeName, nil
	default:
		return "", "", fmt.Errorf(`unrecognized workload "Kind" %v`, workload.Kind)
	}
}

func StatefulSetPodName(appName, podOrdinal string) (string, error) {
	if appName == "" || podOrdinal == "" {
		return "", fmt.Errorf("missing appName or podOrdinal")
	}
	return appName + "-" + podOrdinal, nil
}

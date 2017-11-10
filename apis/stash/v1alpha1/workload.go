package v1alpha1

import (
	"fmt"
	"strings"
)

const (
	AppKindDeployment            = "Deployment"
	AppKindReplicaSet            = "ReplicaSet"
	AppKindReplicationController = "ReplicationController"
	AppKindStatefulSet           = "StatefulSet"
	AppKindDaemonSet             = "DaemonSet"
)

func (workload *LocalTypedReference) Canonicalize() error {
	if workload.Name == "" || workload.Kind == "" {
		return fmt.Errorf("missing workload name or kind")
	}
	switch strings.ToLower(workload.Kind) {
	case "deployments", "deployment", "deploy":
		workload.Kind = AppKindDeployment
	case "replicasets", "replicaset", "rs":
		workload.Kind = AppKindReplicaSet
	case "replicationcontrollers", "replicationcontroller", "rc":
		workload.Kind = AppKindReplicationController
	case "statefulsets", "statefulset":
		workload.Kind = AppKindStatefulSet
	case "daemonsets", "daemonset", "ds":
		workload.Kind = AppKindDaemonSet
	default:
		return fmt.Errorf(`unrecognized workload "Kind" %v`, workload.Kind)
	}
	return nil
}

func (workload LocalTypedReference) HostnamePrefix(podName, nodeName string) (hostname, prefix string, err error) {
	if err := workload.Canonicalize(); err != nil {
		return "", "", err
	}

	if workload.Name == "" || workload.Kind == "" {
		return "", "", fmt.Errorf("missing workload name or kind")
	}
	switch workload.Kind {
	case AppKindDeployment, AppKindReplicaSet, AppKindReplicationController:
		return workload.Name, workload.Kind + "/" + workload.Name, nil
	case AppKindStatefulSet:
		if podName == "" {
			return "", "", fmt.Errorf("missing podName for %s", AppKindStatefulSet)
		}
		return podName, workload.Kind + "/" + podName, nil
	case AppKindDaemonSet:
		if nodeName == "" {
			return "", "", fmt.Errorf("missing nodeName for %s", AppKindDaemonSet)
		}
		return nodeName, workload.Kind + "/" + workload.Name + "/" + nodeName, nil
	default:
		return "", "", fmt.Errorf(`unrecognized workload "Kind" %v`, workload.Kind)
	}
	return
}

func StatefulSetPodName(appName, podOrdinal string) (string, error) {
	if appName == "" || podOrdinal == "" {
		return "", fmt.Errorf("missing appName or podOrdinal")
	}
	return appName + "-" + podOrdinal, nil
}

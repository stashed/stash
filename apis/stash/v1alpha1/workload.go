package v1alpha1

import (
	"fmt"
	"strings"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
)

func (workload *LocalTypedReference) Canonicalize() error {
	if workload.Name == "" || workload.Kind == "" {
		return fmt.Errorf("missing workload name or kind")
	}
	switch strings.ToLower(workload.Kind) {
	case "deployments", "deployment", "deploy":
		workload.Kind = KindDeployment
	case "replicasets", "replicaset", "rs":
		workload.Kind = KindReplicaSet
	case "replicationcontrollers", "replicationcontroller", "rc":
		workload.Kind = KindReplicationController
	case "statefulsets", "statefulset":
		workload.Kind = KindStatefulSet
	case "daemonsets", "daemonset", "ds":
		workload.Kind = KindDaemonSet
	default:
		return fmt.Errorf(`unrecognized workload "Kind" %v`, workload.Kind)
	}
	return nil
}

func (workload LocalTypedReference) GetRepositoryCRDName(podName, nodeName string) string {
	name := ""
	switch workload.Kind {
	case KindDeployment, KindReplicaSet, KindReplicationController:
		name = strings.ToLower(workload.Kind) + "." + workload.Name
	case KindStatefulSet:
		name = strings.ToLower(workload.Kind) + "." + podName
	case KindDaemonSet:
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
	case KindDeployment, KindReplicaSet, KindReplicationController:
		return workload.Name, strings.ToLower(workload.Kind) + "/" + workload.Name, nil
	case KindStatefulSet:
		if podName == "" {
			return "", "", fmt.Errorf("missing podName for %s", KindStatefulSet)
		}
		return podName, strings.ToLower(workload.Kind) + "/" + podName, nil
	case KindDaemonSet:
		if nodeName == "" {
			return "", "", fmt.Errorf("missing nodeName for %s", KindDaemonSet)
		}
		return nodeName, strings.ToLower(workload.Kind) + "/" + workload.Name + "/" + nodeName, nil
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

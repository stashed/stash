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

func ExtractWorkload(workload string) (appKind, appName string, err error) {
	app := strings.SplitN(workload, "/", 2)
	if len(app) != 2 {
		err = fmt.Errorf(`workload must be in the format "Kind/Name", but found %v`, workload)
		return
	}
	appName = app[1]
	switch app[0] {
	case "Deployments", "Deployment", "deployments", "deployment", "deploy":
		appKind = AppKindDeployment
	case "ReplicaSets", "ReplicaSet", "replicasets", "replicaset", "rs":
		appKind = AppKindReplicaSet
	case "ReplicationControllers", "ReplicationController", "replicationcontrollers", "replicationcontroller", "rc":
		appKind = AppKindReplicationController
	case "StatefulSets", "StatefulSet", "statefulsets", "statefulset":
		appKind = AppKindStatefulSet
	case "DaemonSets", "DaemonSet", "daemonsets", "daemonset", "ds":
		appKind = AppKindDaemonSet
	default:
		err = fmt.Errorf(`unrecognized workload "Kind" %v`, app[0])
	}
	return
}

func HostnamePrefixForAppKind(appKind, appName, podName, nodeName string) (hostname, prefix string, err error) {
	switch appKind {
	case AppKindDeployment, AppKindReplicaSet, AppKindReplicationController:
		hostname = appName
	case AppKindStatefulSet:
		hostname = podName
	case AppKindDaemonSet:
		hostname = nodeName + "/" + appName
	default:
		err = fmt.Errorf(`unrecognized workload "Kind" %v`, appKind)
	}
	prefix = appKind + "/" + hostname
	return
}

func StatefulSetPodName(appName, podOrdinal string) (string, error) {
	if appName == "" || podOrdinal == "" {
		return "", fmt.Errorf("missing appName or podOrdinal")
	}
	return appName + "-" + podOrdinal, nil
}

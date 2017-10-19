package v1alpha1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	case "StatefulSets", "StatefulSet":
		appKind = AppKindStatefulSet
	case "DaemonSets", "DaemonSet", "daemonsets", "daemonset":
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

func CheckWorkloadExists(kubeClient kubernetes.Interface, namespace, appKind, appName string) error {
	switch appKind {
	case AppKindDeployment:
		_, err := kubeClient.AppsV1beta1().Deployments(namespace).Get(appName, metav1.GetOptions{})
		if err != nil {
			_, err := kubeClient.ExtensionsV1beta1().Deployments(namespace).Get(appName, metav1.GetOptions{})
			if err != nil {
				fmt.Errorf(`unknown Deployment %s/%s`, namespace, appName)
			}
		}
	case AppKindReplicaSet:
		_, err := kubeClient.ExtensionsV1beta1().ReplicaSets(namespace).Get(appName, metav1.GetOptions{})
		if err != nil {
			fmt.Errorf(`unknown ReplicaSet %s/%s`, namespace, appName)
		}
	case AppKindReplicationController:
		_, err := kubeClient.CoreV1().ReplicationControllers(namespace).Get(appName, metav1.GetOptions{})
		if err != nil {
			fmt.Errorf(`unknown ReplicationController %s/%s`, namespace, appName)
		}
	case AppKindStatefulSet:
		_, err := kubeClient.AppsV1beta1().StatefulSets(namespace).Get(appName, metav1.GetOptions{})
		if err != nil {
			fmt.Errorf(`unknown StatefulSet %s/%s`, namespace, appName)
		}
	case AppKindDaemonSet:
		_, err := kubeClient.ExtensionsV1beta1().DaemonSets(namespace).Get(appName, metav1.GetOptions{})
		if err != nil {
			fmt.Errorf(`unknown DaemonSet %s/%s`, namespace, appName)
		}
	default:
		fmt.Errorf(`unrecognized workload "Kind" %v`, appKind)
	}
	return nil
}

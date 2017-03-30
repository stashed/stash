package dns

import (
	"fmt"
	"os"
	"strings"

	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func GetServiceClusterIP(client clientset.Interface, prefix, hostname string) (string, error) {
	serviceName, namespace, err := splitHostname(hostname)
	if err != nil {
		return "", err
	}
	service, err := client.Core().Services(namespace).Get(serviceName)
	if err != nil {
		return "", err
	}

	sKey := fmt.Sprintf("%s_SERVICE_NAME", strings.ToUpper(prefix))
	os.Setenv(sKey, serviceName)
	nKey := fmt.Sprintf("%s_SERVICE_NAMESPACE", strings.ToUpper(prefix))
	os.Setenv(nKey, namespace)
	return service.Spec.ClusterIP, nil
}

func splitHostname(hostName string) (string, string, error) {
	parts := strings.Split(hostName, ".")
	if len(parts) == 1 {
		return parts[0], kapi.NamespaceDefault, nil
	} else if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf(`Invalid hostname "%v"`, hostName)
}

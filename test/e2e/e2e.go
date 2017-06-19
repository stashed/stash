package test

import (
	"errors"
	"os/user"
	"path/filepath"
	"time"

	"github.com/appscode/log"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/fields"
)

var image = "appscode/restik:latest"

func runController() (*controller.Controller, error) {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(usr.HomeDir, ".kube/config"))
	if err != nil {
		return &controller.Controller{}, err
	}
	kubeClient := clientset.NewForConfigOrDie(config)
	restikClient := rcs.NewForConfigOrDie(config)
	ctrl := controller.NewRestikController(kubeClient, restikClient, image)
	go func() {
		err := ctrl.RunAndHold()
		if err != nil {
			log.Errorln(err)
		}

	}()
	return ctrl, nil
}

func checkEventForBackup(watcher *controller.Controller, objName string) error {
	var err error
	try := 0
	sets := fields.Set{
		"involvedObject.kind":      "Restik",
		"involvedObject.name":      objName,
		"involvedObject.namespace": namespace,
		"type": apiv1.EventTypeNormal,
	}
	fieldSelector := fields.SelectorFromSet(sets)
	for {
		events, err := watcher.Clientset.CoreV1().Events(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
		if err == nil {
			for _, e := range events.Items {
				if e.Reason == controller.EventReasonSuccessfulBackup {
					return nil
				}
			}
		}
		if try > 12 {
			return err
		}
		log.Infoln("Waiting for 10 second for events of backup process")
		time.Sleep(time.Second * 10)
		try++
	}
	return errors.New("Restik backup failed.")
	return err
}

func checkContainerAfterBackupDelete(watcher *controller.Controller, name string, _type string) error {
	try := 0
	var err error
	var containers []apiv1.Container
	for {
		log.Infoln("Waiting 20 sec for checking restik-sidecar deletion")
		time.Sleep(time.Second * 20)
		switch _type {
		case controller.ReplicationController:
			rc, err := watcher.Clientset.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				containers = rc.Spec.Template.Spec.Containers
			}
		case controller.ReplicaSet:
			rs, err := watcher.Clientset.ExtensionsV1beta1().ReplicaSets(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				containers = rs.Spec.Template.Spec.Containers
			}
		case controller.Deployment:
			deployment, err := watcher.Clientset.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				containers = deployment.Spec.Template.Spec.Containers
			}

		case controller.DaemonSet:
			daemonset, err := watcher.Clientset.ExtensionsV1beta1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				containers = daemonset.Spec.Template.Spec.Containers
			}
		}
		err = checkContainerDeletion(containers)
		if err == nil {
			break
		}
		try++
		if try > 6 {
			break
		}
	}
	return err
}

func checkContainerDeletion(containers []apiv1.Container) error {
	for _, c := range containers {
		if c.Name == controller.ContainerName {
			return errors.New("ERROR: Restik sidecar not deleted")
		}
	}
	return nil
}

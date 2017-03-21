package test

import (
	"github.com/appscode/restik/pkg/controller"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/api"
	"time"
	"fmt"
	"errors"
)

func runController() (*controller.Controller, error) {
	config, err :=  clientcmd.BuildConfigFromFlags("", "/home/sauman/.kube/config")
	if err != nil {
		return &controller.Controller{}, err
	}
	controller := controller.New(config, "sauman/restik:test")
	go controller.RunAndHold()
	return controller, nil
}

func checkEventForBackup(watcher *controller.Controller, eventName string) error {
	var err error
	event := &api.Event{}
	try := 0
	for {
		event, err = watcher.Client.Core().Events(namespace).Get(eventName)
		if err == nil {
			break
		}
		if try > 5 {
			return err
		}
		fmt.Println("Waiting for 30 second for events of backup process")
		time.Sleep(time.Second * 30)
		try++
	}
	if event.Reason == "Failed" {
		return errors.New("Restic backup failed.")
	}
	return err
}

func checkDaemonsetAfterBackupDelete(watcher *controller.Controller, daemonset string) error {
	try := 0
	var err error
	for {
		fmt.Println("Waiting 20 sec for checking restik-sedecar deletion...")
		time.Sleep(time.Second * 20)
		d, _ := watcher.Client.Extensions().DaemonSets(namespace).Get(daemonset)
		err = checkContainer(d.Spec.Template.Spec.Containers)
		if err == nil {
			break
		}
		try ++
		if try > 6 {
			break
		}
	}
	return err
}

func checkContainer (containers []api.Container) error {
	for _, c := range containers {
		if c.Name == controller.ContainerName {
			return errors.New("ERROR: Restik sidecar not deleted")
		}
	}
	return nil
}
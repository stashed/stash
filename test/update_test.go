package test

import (
	"fmt"
	"log"
	"testing"

	tcs "github.com/appscode/restik/client/clientset"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func TestBackupUpdate(t *testing.T) {
	factory := cmdutil.NewFactory(nil)
	config, err := factory.ClientConfig()
	if err != nil {
		log.Println(err)
		return
	}
	RestikClient := tcs.NewACRestikForConfigOrDie(config)
	b, err := RestikClient.Restiks("test").Get("testbackup")
	if err != nil {
		fmt.Println(err)
	}
	b.Spec.Schedule = "0 * * * * *"
	b.Spec.RetentionPolicy.KeepLastSnapshots = 5
	b, err = RestikClient.Restiks("test").Update(b)
	if err != nil {
		fmt.Println(err)
	}

}

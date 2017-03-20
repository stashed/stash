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
	extClient := tcs.NewExtensionsForConfigOrDie(config)
	b, err := extClient.Backups("test").Get("saumanbackup")
	if err != nil {
		fmt.Println(err)
	}
	b.Spec.RetentionPolicy.KeepLastSnapshots = 5
	b, err = extClient.Backups("test").Update(b)
	if err != nil {
		fmt.Println(err)
	}

}

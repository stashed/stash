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
	b, err := extClient.Backup("test").Get("saumanbackup")
	if err != nil {
		fmt.Println(err)
	}
	//b.ObjectMeta.Annotations[controller.ImageAnnotation] = "sauman/restik:latest"
	//b.Spec.Tags = append(b.Spec.Tags, "kala")
	var a []string
	b.Spec.RetentionPolicy.KeepTags = a
	b, err = extClient.Backup("test").Update(b)
	if err != nil {
		fmt.Println(err)
	}

}

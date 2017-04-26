package test

import (
	"fmt"
	"log"
	"testing"

	tcs "github.com/appscode/k8s-addons/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func TestBackupUpdate(t *testing.T) {
	factory := cmdutil.NewFactory(nil)
	config, err := factory.ClientConfig()
	if err != nil {
		log.Println(err)
		return
	}
	extClient := tcs.NewACExtensionsForConfigOrDie(config)
	b, err := extClient.Backups("sauman").Get("saumanbackup")
	if err != nil {
		fmt.Println(err)
	}
	b.Spec.Schedule = "0 * * * * *"
	b.Spec.Destination.Volume = api.Volume{
		Name: "newVolume",
		VolumeSource: api.VolumeSource{
			AWSElasticBlockStore: &api.AWSElasticBlockStoreVolumeSource{
				VolumeID: "vol-0acaeb242223da89b",
				FSType:   "ext4",
			},
		},
	}
	b.Spec.RetentionPolicy.KeepLastSnapshots = 5
	b, err = extClient.Backups("sauman").Update(b)
	if err != nil {
		fmt.Println(err)
	}

}

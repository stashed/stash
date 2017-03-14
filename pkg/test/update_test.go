package test

import ("testing"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	tcs "github.com/appscode/restik/client/clientset"
	"log"
	"fmt"
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
	b.Spec.Tags = append(b.Spec.Tags, "sauman")
	b.Spec.RetentionPolicy.SnapshotCount = 10
	b, err = extClient.Backup("test").Update(b)
	if err != nil {
		fmt.Println(err)
	}

}
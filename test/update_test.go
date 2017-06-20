package test

import (
	"fmt"
	"log"
	"testing"

	tcs "github.com/appscode/stash/client/clientset"
	"k8s.io/client-go/tools/clientcmd"
)

func TestBackupUpdate(t *testing.T) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		log.Fatalln(err)
	}
	stashClient := tcs.NewForConfigOrDie(config)
	b, err := stashClient.Restics("test").Get("testbackup")
	if err != nil {
		fmt.Println(err)
	}
	b.Spec.Schedule = "0 * * * * *"
	b.Spec.RetentionPolicy.KeepLastSnapshots = 5
	b, err = stashClient.Restics("test").Update(b)
	if err != nil {
		log.Fatalln(err)
	}
}

package controller

import (
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	//"fmt"
	"fmt"
	tcs "github.com/appscode/k8s-addons/client/clientset"

	//"os"
	//"github.com/ghodss/yaml"
	"k8s.io/kubernetes/pkg/api"
	"testing"
	//"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/fields"
)

func TestC(t *testing.T) {
	factory := cmdutil.NewFactory(nil)
	config, err := factory.ClientConfig()
	if err != nil {
		fmt.Println(err)
	}
		ls := map[string]string{
		"metadata.name": "saumanbackup1",
	}
	c := tcs.NewACExtensionsForConfigOrDie(config)
	bs, err := c.Backups("sauman").List(api.ListOptions{
		//LabelSelector: labels.SelectorFromSet(labels.Set(ls)),
		FieldSelector: fields.SelectorFromSet(fields.Set(ls)),
	})
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range bs.Items {
		fmt.Println(v.Name)
	}
}



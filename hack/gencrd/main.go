package main

import (
	"io/ioutil"
	"os"

	"path/filepath"

	"github.com/appscode/go/log"
	gort "github.com/appscode/go/runtime"
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/kutil/openapi"
	repoinstall "github.com/appscode/stash/apis/repositories/install"
	repov1alpha1 "github.com/appscode/stash/apis/repositories/v1alpha1"
	stashinstall "github.com/appscode/stash/apis/stash/install"
	stashv1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kube-openapi/pkg/common"
)

func generateCRDDefinitions() {
	stashv1alpha1.EnableStatusSubresource = true

	filename := gort.GOPath() + "/src/github.com/appscode/stash/apis/stash/v1alpha1/crds.yaml"
	os.Remove(filename)

	err := os.MkdirAll(filepath.Join(gort.GOPath(), "/src/github.com/appscode/stash/api/crds"), 0755)
	if err != nil {
		log.Fatal(err)
	}

	crds := []*crd_api.CustomResourceDefinition{
		stashv1alpha1.Backup{}.CustomResourceDefinition(),
		stashv1alpha1.Recovery{}.CustomResourceDefinition(),
		stashv1alpha1.Repository{}.CustomResourceDefinition(),
		stashv1alpha1.BackupTemplate{}.CustomResourceDefinition(),
		stashv1alpha1.ContainerTemplate{}.CustomResourceDefinition(),
		stashv1alpha1.BackupTrigger{}.CustomResourceDefinition(),
	}
	for _, crd := range crds {
		filename := filepath.Join(gort.GOPath(), "/src/github.com/appscode/stash/api/crds", crd.Spec.Names.Singular+".yaml")
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		crdutils.MarshallCrd(f, crd, "yaml")
		f.Close()
	}
}
func generateSwaggerJson() {
	var (
		Scheme = runtime.NewScheme()
		Codecs = serializer.NewCodecFactory(Scheme)
	)

	stashinstall.Install(Scheme)
	repoinstall.Install(Scheme)

	apispec, err := openapi.RenderOpenAPISpec(openapi.Config{
		Scheme: Scheme,
		Codecs: Codecs,
		Info: spec.InfoProps{
			Title:   "Stash",
			Version: "v0.8.2",
			Contact: &spec.ContactInfo{
				Name:  "AppsCode Inc.",
				URL:   "https://appscode.com",
				Email: "hello@appscode.com",
			},
			License: &spec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},
		OpenAPIDefinitions: []common.GetOpenAPIDefinitions{
			stashv1alpha1.GetOpenAPIDefinitions,
			repov1alpha1.GetOpenAPIDefinitions,
		},
		Resources: []openapi.TypeInfo{
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralBackup, stashv1alpha1.ResourceKindBackup, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRepository, stashv1alpha1.ResourceKindRepository, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRecovery, stashv1alpha1.ResourceKindRecovery, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralBackupTemplate, stashv1alpha1.ResourceKindBackupTemplate, false},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralContainerTemplate, stashv1alpha1.ResourceKindContainerTemplate, false},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralBackupTrigger, stashv1alpha1.ResourceKindBackupTrigger, true},
		},
		RDResources: []openapi.TypeInfo{
			{repov1alpha1.SchemeGroupVersion, repov1alpha1.ResourcePluralSnapshot, repov1alpha1.ResourceKindSnapshot, true},
		},
	})
	if err != nil {
		glog.Fatal(err)
	}

	filename := gort.GOPath() + "/src/github.com/appscode/stash/api/openapi-spec/swagger.json"
	err = os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		glog.Fatal(err)
	}
	err = ioutil.WriteFile(filename, []byte(apispec), 0644)
	if err != nil {
		glog.Fatal(err)
	}
}

func main() {
	generateCRDDefinitions()
	generateSwaggerJson()
}

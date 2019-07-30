package main

import (
	"io/ioutil"
	"os"

	"path/filepath"

	"github.com/appscode/go/log"
	gort "github.com/appscode/go/runtime"
	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kube-openapi/pkg/common"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	"kmodules.xyz/client-go/openapi"
	"stash.appscode.dev/stash/apis"
	repoinstall "stash.appscode.dev/stash/apis/repositories/install"
	repov1alpha1 "stash.appscode.dev/stash/apis/repositories/v1alpha1"
	stashinstall "stash.appscode.dev/stash/apis/stash/install"
	stashv1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	stashv1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

func generateCRDDefinitions() {
	apis.EnableStatusSubresource = true

	filename := gort.GOPath() + "/src/stash.appscode.dev/stash/apis/stash/v1alpha1/crds.yaml"
	os.Remove(filename)

	path := gort.GOPath() + "/src/stash.appscode.dev/stash/api/crds/"
	os.Remove(filepath.Join(path, "restic.yaml"))
	os.Remove(filepath.Join(path, "recovery.yaml"))
	os.Remove(filepath.Join(path, "repository.yaml"))

	// generate "v1alpha1" crds
	v1alpha1CRDs := []*crd_api.CustomResourceDefinition{
		stashv1alpha1.Restic{}.CustomResourceDefinition(),
		stashv1alpha1.Recovery{}.CustomResourceDefinition(),
		stashv1alpha1.Repository{}.CustomResourceDefinition(),
	}
	genCRD(stashv1alpha1.SchemeGroupVersion.Version, v1alpha1CRDs)

	// generate "v1beta1" crds
	v1beta1CRDs := []*crd_api.CustomResourceDefinition{
		stashv1beta1.Function{}.CustomResourceDefinition(),
		stashv1beta1.BackupConfiguration{}.CustomResourceDefinition(),
		stashv1beta1.BackupSession{}.CustomResourceDefinition(),
		stashv1beta1.BackupBlueprint{}.CustomResourceDefinition(),
		stashv1beta1.RestoreSession{}.CustomResourceDefinition(),
		stashv1beta1.Task{}.CustomResourceDefinition(),
	}
	genCRD(stashv1beta1.SchemeGroupVersion.Version, v1beta1CRDs)

}

func genCRD(version string, crds []*crd_api.CustomResourceDefinition) {

	err := os.MkdirAll(filepath.Join(gort.GOPath(), "/src/stash.appscode.dev/stash/api/crds", version), 0755)
	if err != nil {
		log.Fatal(err)
	}

	for _, crd := range crds {
		filename := filepath.Join(gort.GOPath(), "/src/stash.appscode.dev/stash/api/crds", version, crd.Spec.Names.Singular+".yaml")
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
			Version: "v0.8.3",
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
			stashv1beta1.GetOpenAPIDefinitions,
			repov1alpha1.GetOpenAPIDefinitions,
		},
		Resources: []openapi.TypeInfo{
			// v1alpha1 resources
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRestic, stashv1alpha1.ResourceKindRestic, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRepository, stashv1alpha1.ResourceKindRepository, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRecovery, stashv1alpha1.ResourceKindRecovery, true},

			// v1beta1 resources
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralBackupConfiguration, stashv1beta1.ResourceKindBackupConfiguration, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindBackupSession, stashv1beta1.ResourceKindBackupSession, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindBackupBlueprint, stashv1beta1.ResourceKindBackupBlueprint, false},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralRestoreSession, stashv1beta1.ResourceKindRestoreSession, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindFunction, stashv1beta1.ResourceKindFunction, false},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralTask, stashv1beta1.ResourceKindTask, false},
		},
		RDResources: []openapi.TypeInfo{
			{repov1alpha1.SchemeGroupVersion, repov1alpha1.ResourcePluralSnapshot, repov1alpha1.ResourceKindSnapshot, true},
		},
	})
	if err != nil {
		glog.Fatal(err)
	}

	filename := gort.GOPath() + "/src/stash.appscode.dev/stash/api/openapi-spec/swagger.json"
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
	//generateSwaggerJson()
}

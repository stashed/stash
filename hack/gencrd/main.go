package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	repoinstall "stash.appscode.dev/stash/apis/repositories/install"
	repov1alpha1 "stash.appscode.dev/stash/apis/repositories/v1alpha1"
	stashinstall "stash.appscode.dev/stash/apis/stash/install"
	stashv1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	stashv1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"

	gort "github.com/appscode/go/runtime"
	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kube-openapi/pkg/common"
	"kmodules.xyz/client-go/openapi"
)

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
			Version: "v0.9.0-rc.0",
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
		//nolint:govet
		Resources: []openapi.TypeInfo{
			// v1alpha1 resources
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRestic, stashv1alpha1.ResourceKindRestic, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRepository, stashv1alpha1.ResourceKindRepository, true},
			{stashv1alpha1.SchemeGroupVersion, stashv1alpha1.ResourcePluralRecovery, stashv1alpha1.ResourceKindRecovery, true},

			// v1beta1 resources
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralBackupConfiguration, stashv1beta1.ResourceKindBackupConfiguration, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindBackupSession, stashv1beta1.ResourceKindBackupSession, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindBackupBatch, stashv1beta1.ResourceKindBackupBatch, false},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindBackupBlueprint, stashv1beta1.ResourceKindBackupBlueprint, false},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralRestoreSession, stashv1beta1.ResourceKindRestoreSession, true},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourceKindFunction, stashv1beta1.ResourceKindFunction, false},
			{stashv1beta1.SchemeGroupVersion, stashv1beta1.ResourcePluralTask, stashv1beta1.ResourceKindTask, false},
		},
		//nolint:govet
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
	generateSwaggerJson()
}

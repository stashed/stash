/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"io"
	"net"

	"stash.appscode.dev/apimachinery/apis/repositories/v1alpha1"
	"stash.appscode.dev/stash/pkg/controller"
	"stash.appscode.dev/stash/pkg/server"

	"github.com/spf13/pflag"
	licenseEnforcer "go.bytebuilders.dev/license-verifier/kubernetes"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clientcmd"
)

const defaultEtcdPathPrefix = "/registry/stash.appscode.com"

type StashOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	ExtraOptions       *ExtraOptions

	StdOut io.Writer
	StdErr io.Writer
}

func NewStashOptions(out, errOut io.Writer) *StashOptions {
	o := &StashOptions{
		// TODO we will nil out the etcd storage options.  This requires a later level of k8s.io/apiserver
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			server.Codecs.LegacyCodec(admissionv1beta1.SchemeGroupVersion),
			genericoptions.NewProcessInfo("stash-operator", meta.Namespace()),
		),
		ExtraOptions: NewExtraOptions(),
		StdOut:       out,
		StdErr:       errOut,
	}
	o.RecommendedOptions.Etcd = nil
	o.RecommendedOptions.Admission = nil

	return o
}

func (o StashOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.ExtraOptions.AddFlags(fs)
}

func (o StashOptions) Validate(args []string) error {
	var errs []error
	errs = append(errs, o.RecommendedOptions.Validate()...)
	errs = append(errs, o.ExtraOptions.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *StashOptions) Complete() error {
	return nil
}

func (o StashOptions) Config() (*server.StashConfig, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(server.Codecs)
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	// Fixes https://github.com/Azure/AKS/issues/522
	clientcmd.Fix(serverConfig.ClientConfig)

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(v1alpha1.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(server.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "stash-operator"
	serverConfig.OpenAPIConfig.Info.Version = v1alpha1.SchemeGroupVersion.Version
	serverConfig.OpenAPIConfig.IgnorePrefixes = []string{
		"/swaggerapi",
		"/apis/admission.stash.appscode.com/v1alpha1",
		"/apis/admission.stash.appscode.com/v1alpha1/resticvalidators",
		"/apis/admission.stash.appscode.com/v1alpha1/recoveryvalidators",
		"/apis/admission.stash.appscode.com/v1alpha1/repositoryvalidators",
		"/apis/admission.stash.appscode.com/v1alpha1/deploymentmutators",
		"/apis/admission.stash.appscode.com/v1alpha1/daemonsetmutators",
		"/apis/admission.stash.appscode.com/v1alpha1/statefulsetmutators",
		"/apis/admission.stash.appscode.com/v1alpha1/replicationcontrollermutators",
		"/apis/admission.stash.appscode.com/v1alpha1/replicasetmutators",
		"/apis/admission.stash.appscode.com/v1alpha1/deploymentconfigmutators",
		"/apis/admission.stash.appscode.com/v1beta1/restoresessionvalidators",
		"/apis/admission.stash.appscode.com/v1beta1/restoresessionmutators",
	}

	extraConfig := controller.NewConfig(serverConfig.ClientConfig)
	if err := o.ExtraOptions.ApplyTo(extraConfig); err != nil {
		return nil, err
	}

	config := &server.StashConfig{
		GenericConfig: serverConfig,
		ExtraConfig:   extraConfig,
	}
	return config, nil
}

func (o StashOptions) Run(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	s, err := config.Complete().New()
	if err != nil {
		return err
	}

	// Start periodic license verification
	//nolint:errcheck
	go licenseEnforcer.VerifyLicensePeriodically(config.ExtraConfig.ClientConfig, o.ExtraOptions.LicenseFile, stopCh)

	return s.Run(stopCh)
}

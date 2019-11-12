/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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
	"os"
	"strings"
	"syscall"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/repositories/v1alpha1"
	"stash.appscode.dev/stash/pkg/controller"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/server"

	"github.com/appscode/go/log"
	"github.com/spf13/pflag"
	license_client "go.bytebuilders.dev/client-go"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/core"
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

	// Run initial license verification. Don't start any controller if licence is invalid
	err = verifyLicense(config.ExtraConfig.KubeClient)
	if err != nil {
		return handleLicenseVerificationFailure(config.ExtraConfig.KubeClient, err)
	}

	// Start periodic license verification
	go runPeriodicLicenseVerification(config.ExtraConfig.KubeClient, stopCh)

	return s.Run(stopCh)
}

func runPeriodicLicenseVerification(kubeClient kubernetes.Interface, stopCh <-chan struct{}) {
	ticker := time.NewTicker(apis.LicenseVerificationInterval)
	for range ticker.C {
		err := verifyLicense(kubeClient)
		if err != nil {
			_ = handleLicenseVerificationFailure(kubeClient, err)
			// send interrupt so that all go-routine shut-down
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			return
		}
	}
	<-stopCh
}

func verifyLicense(kubeClient kubernetes.Interface) error {
	// In order to verify license, we need to know clusterID, productID and productOwnerID
	// in addition to license key itself.
	// clusterID is the UID of the kube-system namespace.
	// productID and productOwnerID are unique id provided to a product while registering the product
	// in the byte builders marketplace. They should be hardcoded into a product source code as constant

	log.Infoln("Verifying license.....")
	ns, err := kubeClient.CoreV1().Namespaces().Get("kube-system", metav1.GetOptions{})
	if err != nil {
		return err
	}

	clusterID := string(ns.UID)
	licenseClient := license_client.NewClient(
		"",
		strings.TrimSuffix(os.Getenv(apis.StashLicenseKey), "\n"),
		"https://appscode.ninja",
	)

	_, err = licenseClient.GetLicensePlan(clusterID, apis.StashProductID, apis.StashOwnerID)
	if err != nil {
		return err
	}
	log.Infoln("License has been verified successfully")
	return nil
}

func handleLicenseVerificationFailure(kubeClient kubernetes.Interface, licenseErr error) error {
	// Log license verification failure
	log.Warningln("failed to verify license. Reason: ", licenseErr.Error())

	// Write event to stash operator deployment
	podName := os.Getenv("MY_POD_NAME")
	namespace := os.Getenv("MY_POD_NAMESPACE")

	parts := strings.Split(podName, "-")
	operatorName := strings.Join(parts[0:len(parts)-2], "-")

	dpl, err := kubeClient.AppsV1().Deployments(namespace).Get(operatorName, metav1.GetOptions{})
	if err != nil {
		return errors.NewAggregate([]error{licenseErr, err})
	}
	_, err = eventer.CreateEvent(
		kubeClient,
		eventer.EventSourceLicenseVerifier,
		dpl,
		core.EventTypeWarning,
		eventer.EventReasonLicenseVerificationFailed,
		fmt.Sprintf("filed to verify license. Reason: %s", licenseErr.Error()),
	)
	return errors.NewAggregate([]error{licenseErr, err})
}

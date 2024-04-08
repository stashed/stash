/*
Copyright AppsCode Inc.

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

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"go.bytebuilders.dev/license-verifier/apis/licenses/v1alpha1"
	"go.bytebuilders.dev/license-verifier/info"

	"github.com/pkg/errors"
	proxyserver "go.bytebuilders.dev/license-proxyserver/apis/proxyserver/v1alpha1"
	proxyclient "go.bytebuilders.dev/license-proxyserver/client/clientset/versioned"
	verifier "go.bytebuilders.dev/license-verifier"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/client-go/kubernetes"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog/v2"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/dynamic"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clusterid"
)

const (
	EventSourceLicenseVerifier           = "License Verifier"
	EventReasonLicenseVerificationFailed = "License Verification Failed"

	licensePath          = "/appscode/license"
	licenseCheckInterval = 1 * time.Hour
)

type LicenseEnforcer struct {
	licenseFile string
	opts        verifier.VerifyOptions
	config      *rest.Config
	kc          kubernetes.Interface
}

// NewLicenseEnforcer returns a newly created license enforcer
func NewLicenseEnforcer(config *rest.Config, licenseFile string) (*LicenseEnforcer, error) {
	le := LicenseEnforcer{
		config:      config,
		licenseFile: licenseFile,
		opts: verifier.VerifyOptions{
			Features: info.ProductName,
		},
	}

	caData, err := info.LoadLicenseCA()
	if err != nil {
		return &le, err
	}
	le.opts.CACert, err = info.ParseCertificate(caData)
	if err != nil {
		return &le, err
	}
	return &le, nil
}

func MustLicenseEnforcer(config *rest.Config, licenseFile string) *LicenseEnforcer {
	le, err := NewLicenseEnforcer(config, licenseFile)
	if err != nil {
		panic("failed to instantiate license enforcer, err:" + err.Error())
	}
	return le
}

func (le *LicenseEnforcer) getLicense() ([]byte, error) {
	licenseBytes, err := os.ReadFile(le.licenseFile)
	if errors.Is(err, os.ErrNotExist) || (err == nil && le.invalidLicense(licenseBytes)) {
		req := proxyserver.LicenseRequest{
			TypeMeta: metav1.TypeMeta{},
			Request: &proxyserver.LicenseRequestRequest{
				Features: info.Features(),
			},
		}
		pc, err := proxyclient.NewForConfig(le.config)
		if err != nil {
			return nil, errors.Wrap(err, "failed create client for license-proxyserver")
		}
		resp, err := pc.ProxyserverV1alpha1().LicenseRequests().Create(context.TODO(), &req, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to read license")
		}
		licenseBytes = []byte(resp.Response.License)
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to read license")
	}
	return licenseBytes, nil
}

func (le *LicenseEnforcer) invalidLicense(license []byte) bool {
	le.opts.License = license
	// We don't want to acquire license from license-proxyserver is the license file
	// contains a valid license for a different product.
	// We want to acquire license-proxyserver is a previously valid license has not expired.
	// So, we don't check features in the license found is license file.
	l, err := verifier.ParseLicense(le.opts.ParserOptions)
	return sets.NewString(l.Features...).HasAny(info.ParseFeatures(le.opts.Features)...) && err != nil
}

func (le *LicenseEnforcer) createClients() (err error) {
	if le.kc == nil {
		le.kc, err = kubernetes.NewForConfig(le.config)
	}
	return err
}

func (le *LicenseEnforcer) acquireLicense() (err error) {
	le.opts.License, err = le.getLicense()
	return err
}

func (le *LicenseEnforcer) readClusterUID() (err error) {
	if le.opts.ClusterUID != "" {
		return
	}
	le.opts.ClusterUID, err = clusterid.ClusterUID(le.kc.CoreV1().Namespaces())
	return err
}

func (le *LicenseEnforcer) handleLicenseVerificationFailure(licenseErr error) error {
	// Send interrupt so that all go-routines shut-down gracefully
	// https://pracucci.com/graceful-shutdown-of-kubernetes-pods.html
	// https://linuxhandbook.com/sigterm-vs-sigkill/
	// https://pracucci.com/graceful-shutdown-of-kubernetes-pods.html
	//nolint:errcheck
	defer func() {
		// Need to send signal twice because
		// we catch the first INT/TERM signal
		// ref: https://github.com/kubernetes/apiserver/blob/8d97c871d91c75b81b8b4c438f4dd1eaa7f35052/pkg/server/signal.go#L47-L51
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		time.Sleep(30 * time.Second)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
	}()

	// Log licenseInfo verification failure
	klog.Errorln("Failed to verify license. Reason: ", licenseErr.Error())

	// Read the namespace of current pod
	namespace := meta.PodNamespace()

	// Find the root owner of this pod
	owner, _, err := dynamic.DetectWorkload(
		context.TODO(),
		le.config,
		core.SchemeGroupVersion.WithResource(core.ResourcePods.String()),
		namespace,
		meta.PodName(),
	)
	if err != nil {
		return err
	}
	ref, err := reference.GetReference(clientscheme.Scheme, owner)
	if err != nil {
		return err
	}
	eventMeta := metav1.ObjectMeta{
		Name:      meta.NameWithSuffix(owner.GetName(), "license"),
		Namespace: namespace,
	}
	// Create an event against the root owner specifying that the license verification failed
	_, _, err = core_util.CreateOrPatchEvent(context.TODO(), le.kc, eventMeta, func(in *core.Event) *core.Event {
		in.InvolvedObject = *ref
		in.Type = core.EventTypeWarning
		in.Source = core.EventSource{Component: EventSourceLicenseVerifier}
		in.Reason = EventReasonLicenseVerificationFailed
		in.Message = fmt.Sprintf("Failed to verify license. Reason: %s", licenseErr.Error())

		if in.FirstTimestamp.IsZero() {
			in.FirstTimestamp = metav1.Now()
		}
		in.LastTimestamp = metav1.Now()
		in.Count = in.Count + 1

		return in
	}, metav1.PatchOptions{})
	return err
}

// Install adds the License info handler
func (le *LicenseEnforcer) Install(c *mux.PathRecorderMux) {
	// Create Kubernetes client
	err := le.createClients()
	if err != nil {
		klog.Fatal(err)
		return
	}
	c.Handle(licensePath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-content-type-options", "nosniff")

		utilruntime.Must(json.NewEncoder(w).Encode(le.LoadLicense()))
	}))
}

func (le *LicenseEnforcer) LoadLicense() v1alpha1.License {
	utilruntime.Must(le.createClients())

	// Read cluster UID (UID of the "kube-system" namespace)
	err := le.readClusterUID()
	if err != nil {
		license, _ := verifier.BadLicense(err)
		return license
	}
	// Read license from file
	err = le.acquireLicense()
	if err != nil {
		license, _ := verifier.BadLicense(err)
		return license
	}
	license, _ := verifier.CheckLicense(le.opts)
	return license
}

// VerifyLicensePeriodically periodically verifies whether the provided license is valid for the current cluster or not.
func VerifyLicensePeriodically(config *rest.Config, licenseFile string, stopCh <-chan struct{}) error {
	if info.SkipLicenseVerification() {
		klog.Infoln("License verification skipped")
		return nil
	}

	le, err := NewLicenseEnforcer(config, licenseFile)
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	if err := verifyLicensePeriodically(le, licenseFile, stopCh); err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	return nil
}

func verifyLicensePeriodically(le *LicenseEnforcer, licenseFile string, stopCh <-chan struct{}) error {
	// Create Kubernetes client
	err := le.createClients()
	if err != nil {
		return err
	}
	// Read cluster UID (UID of the "kube-system" namespace)
	err = le.readClusterUID()
	if err != nil {
		return err
	}

	// Periodically verify license with 1 hour interval
	fn := func() (done bool, err error) {
		klog.V(8).Infoln("Verifying license.......")
		// Read license from file
		err = le.acquireLicense()
		if err != nil {
			return false, err
		}
		// Validate license
		_, err = verifier.CheckLicense(le.opts)
		if err != nil {
			return false, err
		}
		klog.Infoln("Successfully verified license!")
		// return false so that the loop never ends
		return false, nil
	}

	return wait.PollImmediateUntil(licenseCheckInterval, fn, stopCh)
}

// CheckLicenseFile verifies whether the provided license is valid for the current cluster or not.
func CheckLicenseFile(config *rest.Config, licenseFile string) error {
	if info.SkipLicenseVerification() {
		klog.Infoln("License verification skipped")
		return nil
	}

	klog.V(8).Infoln("Verifying license.......")
	le, err := NewLicenseEnforcer(config, licenseFile)
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	if err := checkLicenseFile(le); err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	return nil
}

func checkLicenseFile(le *LicenseEnforcer) error {
	// Create Kubernetes client
	err := le.createClients()
	if err != nil {
		return err
	}
	// Read cluster UID (UID of the "kube-system" namespace)
	err = le.readClusterUID()
	if err != nil {
		return err
	}
	// Read license from file
	err = le.acquireLicense()
	if err != nil {
		return err
	}
	// Validate license
	_, err = verifier.CheckLicense(le.opts)
	if err != nil {
		return err
	}
	klog.Infoln("Successfully verified license!")
	return nil
}

// CheckLicenseEndpoint verifies whether the provided api server has a valid license is valid for features.
func CheckLicenseEndpoint(config *rest.Config, apiServiceName string, features []string) error {
	aggrClient, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	apiSvc, err := aggrClient.ApiregistrationV1().APIServices().Get(context.TODO(), apiServiceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	c2 := *config
	c2.CAData = apiSvc.Spec.CABundle
	c2.Insecure = apiSvc.Spec.InsecureSkipTLSVerify
	rt, err := rest.TransportFor(&c2)
	if err != nil {
		return err
	}
	hc := http.Client{
		Transport: rt,
		Timeout:   30 * time.Second,
	}

	u, err := url.Parse(fmt.Sprintf("https://%s.%s.svc", apiSvc.Spec.Service.Name, apiSvc.Spec.Service.Namespace))
	if err != nil {
		return err
	}
	u.Path = licensePath

	resp, err := hc.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var license v1alpha1.License
	err = json.Unmarshal(data, &license)
	if err != nil {
		return err
	}

	if license.Status != v1alpha1.LicenseActive {
		return fmt.Errorf("license %s is not active, status: %s, reason: %s", license.ID, license.Status, license.Reason)
	}

	if !sets.NewString(license.Features...).HasAny(features...) {
		return fmt.Errorf("license %s is not valid for products %q", license.ID, strings.Join(features, ","))
	}
	return nil
}

func LicenseProvided(cfg *rest.Config, licenseFile string) bool {
	if licenseFile != "" {
		return true
	}

	if cfg != nil {
		ok, _ := discovery.HasGVK(
			kubernetes.NewForConfigOrDie(cfg).Discovery(),
			proxyserver.SchemeGroupVersion.String(),
			proxyserver.ResourceKindLicenseRequest)
		return ok
	}
	return false
}

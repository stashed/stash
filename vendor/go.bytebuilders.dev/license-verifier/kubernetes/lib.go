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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	verifier "go.bytebuilders.dev/license-verifier"
	"go.bytebuilders.dev/license-verifier/info"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/dynamic"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clusterid"
)

const (
	EventSourceLicenseVerifier           = "License Verifier"
	EventReasonLicenseVerificationFailed = "License Verification Failed"
)

type LicenseEnforcer struct {
	opts        *verifier.Options
	config      *rest.Config
	k8sClient   kubernetes.Interface
	licenseFile string
}

// VerifyLicensePeriodically periodically verifies whether the provided license is valid for the current cluster or not.
func VerifyLicensePeriodically(config *rest.Config, licenseFile string, stopCh <-chan struct{}) error {
	if info.SkipLicenseVerification() {
		klog.Infoln("License verification skipped")
		return nil
	}

	le := &LicenseEnforcer{
		licenseFile: licenseFile,
		config:      config,
		opts: &verifier.Options{
			CACert:      []byte(info.LicenseCA),
			ProductName: info.ProductName,
		},
	}
	// Create Kubernetes client
	err := le.createClients()
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	// Read cluster UID (UID of the "kube-system" namespace)
	err = le.readClusterUID()
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}

	// Periodically verify license with 1 hour interval
	return wait.PollImmediateUntil(1*time.Hour, func() (done bool, err error) {
		klog.V(8).Infoln("Verifying license.......")
		// Read license from file
		err = le.readLicenseFromFile()
		if err != nil {
			return false, le.handleLicenseVerificationFailure(err)
		}
		// Validate license
		err = verifier.VerifyLicense(le.opts)
		if err != nil {
			return false, le.handleLicenseVerificationFailure(err)
		}
		klog.Infoln("Successfully verified license!")
		// return false so that the loop never ends
		return false, nil
	}, stopCh)
}

// VerifyLicense verifies whether the provided license is valid for the current cluster or not.
func VerifyLicense(config *rest.Config, licenseFile string) error {
	if info.SkipLicenseVerification() {
		klog.Infoln("License verification skipped")
		return nil
	}

	klog.V(8).Infoln("Verifying license.......")
	le := &LicenseEnforcer{
		licenseFile: licenseFile,
		config:      config,
		opts: &verifier.Options{
			CACert:      []byte(info.LicenseCA),
			ProductName: info.ProductName,
		},
	}
	// Create Kubernetes client
	err := le.createClients()
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	// Read cluster UID (UID of the "kube-system" namespace)
	err = le.readClusterUID()
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	// Read license from file
	err = le.readLicenseFromFile()
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	// Validate license
	err = verifier.VerifyLicense(le.opts)
	if err != nil {
		return le.handleLicenseVerificationFailure(err)
	}
	klog.Infoln("Successfully verified license!")
	return nil
}

func (le *LicenseEnforcer) createClients() (err error) {
	le.k8sClient, err = kubernetes.NewForConfig(le.config)
	return err
}

func (le *LicenseEnforcer) readLicenseFromFile() (err error) {
	le.opts.License, err = ioutil.ReadFile(le.licenseFile)
	return err
}

func (le *LicenseEnforcer) readClusterUID() (err error) {
	le.opts.ClusterUID, err = clusterid.ClusterUID(le.k8sClient.CoreV1().Namespaces())
	return err
}

func (le *LicenseEnforcer) podName() (string, error) {
	if name, ok := os.LookupEnv("MY_POD_NAME"); ok {
		return name, nil
	}

	if meta.PossiblyInCluster() {
		// Read current pod name
		return os.Hostname()
	}
	return "", errors.New("failed to detect pod name")
}

func (le *LicenseEnforcer) handleLicenseVerificationFailure(licenseErr error) error {
	// Send interrupt so that all go-routines shut-down gracefully
	//nolint:errcheck
	defer syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	// Log licenseInfo verification failure
	klog.Errorln("Failed to verify license. Reason: ", licenseErr.Error())

	podName, err := le.podName()
	if err != nil {
		return err
	}
	// Read the namespace of current pod
	namespace := meta.Namespace()

	// Find the root owner of this pod
	owner, _, err := dynamic.DetectWorkload(
		context.TODO(),
		le.config,
		core.SchemeGroupVersion.WithResource(core.ResourcePods.String()),
		namespace,
		podName,
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
	_, _, err = core_util.CreateOrPatchEvent(context.TODO(), le.k8sClient, eventMeta, func(in *core.Event) *core.Event {
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

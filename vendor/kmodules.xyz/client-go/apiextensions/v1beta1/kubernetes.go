package v1beta1

import (
	"fmt"
	"net/http"
	"time"

	discovery_util "kmodules.xyz/client-go/discovery"

	"github.com/pkg/errors"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func RegisterCRDs(disClient discovery.DiscoveryInterface, apiextClient crd_cs.ApiextensionsV1beta1Interface, crds []*crd_api.CustomResourceDefinition) error {
	major, minor, _, _, _, err := discovery_util.GetVersionInfo(disClient)
	if err != nil {
		return err
	}

	for _, crd := range crds {
		if major == 1 && minor <= 11 {
			// CRD schema must only have "properties", "required" or "description" at the root if the status subresource is enabled
			// xref: https://github.com/stashed/stash/issues/1007#issuecomment-570888875
			crd.Spec.Validation.OpenAPIV3Schema.Type = ""
		}

		existing, err := apiextClient.CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = apiextClient.CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			// Update AdditionalPrinterColumns, Catagories, ShortNames, Validation
			// and Subresources of existing CRD.
			existing.Spec.AdditionalPrinterColumns = crd.Spec.AdditionalPrinterColumns
			existing.Spec.Names.Categories = crd.Spec.Names.Categories
			existing.Spec.Names.ShortNames = crd.Spec.Names.ShortNames
			existing.Spec.Validation = crd.Spec.Validation

			if crd.Spec.Subresources != nil && existing.Spec.Subresources == nil {
				existing.Spec.Subresources = &crd_api.CustomResourceSubresources{}
				if crd.Spec.Subresources.Status != nil && existing.Spec.Subresources.Status == nil {
					existing.Spec.Subresources.Status = crd.Spec.Subresources.Status
				}
				if crd.Spec.Subresources.Scale != nil && existing.Spec.Subresources.Scale == nil {
					existing.Spec.Subresources.Scale = crd.Spec.Subresources.Scale
				}
			} else if crd.Spec.Subresources == nil && existing.Spec.Subresources != nil {
				existing.Spec.Subresources = nil
			}
			_, err = apiextClient.CustomResourceDefinitions().Update(existing)
			if err != nil {
				return err
			}
		}
	}
	return WaitForCRDReady(apiextClient.RESTClient(), crds)
}

func WaitForCRDReady(restClient rest.Interface, crds []*crd_api.CustomResourceDefinition) error {
	err := wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		for _, crd := range crds {
			res := restClient.Get().AbsPath("apis", crd.Spec.Group, crd.Spec.Versions[0].Name, crd.Spec.Names.Plural).Do()
			err := res.Error()
			if err != nil {
				// RESTClient returns *apierrors.StatusError for any status codes < 200 or > 206
				// and http.Client.Do errors are returned directly.
				if se, ok := err.(*kerr.StatusError); ok {
					if se.Status().Code == http.StatusNotFound {
						return false, nil
					}
				}
				return false, err
			}

			var statusCode int
			res.StatusCode(&statusCode)
			if statusCode != http.StatusOK {
				return false, errors.Errorf("invalid status code: %d", statusCode)
			}
		}

		return true, nil
	})

	return errors.Wrap(err, fmt.Sprintf("timed out waiting for CRD"))
}

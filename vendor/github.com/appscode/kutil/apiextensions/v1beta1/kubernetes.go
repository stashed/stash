package v1beta1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

func RegisterCRDs(client crd_cs.ApiextensionsV1beta1Interface, crds []*crd_api.CustomResourceDefinition) error {
	for _, crd := range crds {
		existing, err := client.CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = client.CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			existing.Spec.Validation = crd.Spec.Validation
			_, err = client.CustomResourceDefinitions().Update(existing)
			if err != nil {
				return err
			}
		}
	}
	return WaitForCRDReady(client.RESTClient(), crds)
}

func WaitForCRDReady(restClient rest.Interface, crds []*crd_api.CustomResourceDefinition) error {
	err := wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		for _, crd := range crds {
			res := restClient.Get().AbsPath("apis", crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Plural).Do()
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

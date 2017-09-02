package migrator

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/log"
	sapi "github.com/appscode/stash/apis/stash"
	sapi_v1alpha1 "github.com/appscode/stash/apis/stash"
	"github.com/appscode/stash/pkg/util"
	"github.com/hashicorp/go-version"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type migrationState struct {
	tprRegDeleted bool
	crdCreated    bool
}

type migrator struct {
	kubeClient       clientset.Interface
	apiExtKubeClient apiextensionsclient.Interface

	migrationState *migrationState
}

func NewMigrator(kubeClient clientset.Interface, apiExtKubeClient apiextensionsclient.Interface) *migrator {
	return &migrator{
		migrationState:   &migrationState{},
		kubeClient:       kubeClient,
		apiExtKubeClient: apiExtKubeClient,
	}
}

func (m *migrator) isMigrationNeeded() (bool, error) {
	v, err := m.kubeClient.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}

	ver, err := version.NewVersion(v.String())
	if err != nil {
		return false, err
	}

	mv := ver.Segments()[1]

	if mv == 7 {
		_, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(
			sapi_v1alpha1.ResourceNameRestic+"."+sapi_v1alpha1.SchemeGroupVersion.Group,
			metav1.GetOptions{},
		)
		if err != nil {
			if !kerr.IsNotFound(err) {
				return false, err
			}
		} else {
			return true, nil
		}
	}

	return false, nil
}

func (m *migrator) RunMigration() error {
	needed, err := m.isMigrationNeeded()
	if err != nil {
		return err
	}

	if needed {
		if err := m.migrateTPR2CRD(); err != nil {
			return m.rollback()
		}
	}

	return nil
}

func (m *migrator) migrateTPR2CRD() error {
	log.Debugln("Performing TPR to CRD migration.")

	log.Debugln("Deleting TPRs.")
	if err := m.deleteTPRs(); err != nil {
		return errors.New("Failed to Delete TPRs")
	}

	m.migrationState.tprRegDeleted = true

	log.Debugln("Creating CRDs.")
	if err := m.createCRDs(); err != nil {
		return errors.New("Failed to create CRDs")
	}

	m.migrationState.crdCreated = true
	return nil
}

func (m *migrator) deleteTPRs() error {
	tprClient := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources()

	deleteTPR := func(resourceName string) error {
		name := resourceName + "." + sapi_v1alpha1.SchemeGroupVersion.Group
		if err := tprClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to remove %s TPR", name)
		}
		return nil
	}

	if err := deleteTPR(sapi_v1alpha1.ResourceNameRestic); err != nil {
		return err
	}

	return nil
}

func (m *migrator) createCRDs() error {
	crds := []*apiextensions.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   sapi_v1alpha1.ResourceTypeRestic + "." + sapi_v1alpha1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   sapi.GroupName,
				Version: sapi_v1alpha1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   sapi_v1alpha1.ResourceNameRestic,
					Plural:     sapi_v1alpha1.ResourceTypeRestic,
					Kind:       sapi_v1alpha1.ResourceKindRestic,
					ShortNames: []string{"rst"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return util.WaitForCRDReady(
		m.kubeClient.CoreV1().RESTClient(),
		crds,
	)
}

func (m *migrator) rollback() error {
	log.Debugln("Rolling back migration.")

	ms := m.migrationState

	if ms.crdCreated {
		log.Debugln("Deleting CRDs.")
		err := m.deleteCRDs()
		if err != nil {
			return errors.New("Failed to delete CRDs")
		}
	}

	if ms.tprRegDeleted {
		log.Debugln("Creating TPRs.")
		err := m.createTPRs()
		if err != nil {
			return errors.New("Failed to recreate TPRs")
		}

		err = m.waitForTPRsReady()
		if err != nil {
			return errors.New("Failed to be ready TPRs")
		}
	}

	return nil
}

func (m *migrator) deleteCRDs() error {
	crdClient := m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions()

	deleteCRD := func(resourceType string) error {
		name := resourceType + "." + sapi_v1alpha1.SchemeGroupVersion.Group
		err := crdClient.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf(`Failed to delete CRD "%s""`, name)
		}
		return nil
	}

	if err := deleteCRD(sapi_v1alpha1.ResourceTypeRestic); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPRs() error {
	if err := m.createTPR(sapi_v1alpha1.ResourceNameRestic); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPR(resourceName string) error {
	name := resourceName + "." + sapi_v1alpha1.SchemeGroupVersion.Group
	_, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(name, metav1.GetOptions{})
	if !kerr.IsNotFound(err) {
		return err
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "stash",
			},
		},
		Versions: []extensions.APIVersion{
			{
				Name: sapi_v1alpha1.SchemeGroupVersion.Version,
			},
		},
	}

	_, err = m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Create(thirdPartyResource)
	return err
}

func (m *migrator) waitForTPRsReady() error {
	labelMap := map[string]string{
		"app": "stash",
	}

	return wait.Poll(3*time.Second, 10*time.Minute, func() (bool, error) {
		crdList, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labelMap).String(),
		})
		if err != nil {
			return false, err
		}

		if len(crdList.Items) == 3 {
			return true, nil
		}

		return false, nil
	})
}

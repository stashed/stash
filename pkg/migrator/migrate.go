package migrator

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/hashicorp/go-version"
	extensions "k8s.io/api/extensions/v1beta1"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type migrationState struct {
	tprRegDeleted bool
	crdCreated    bool
}

type migrator struct {
	kubeClient       kubernetes.Interface
	apiExtKubeClient crd_cs.ApiextensionsV1beta1Interface

	migrationState *migrationState
}

func NewMigrator(kubeClient kubernetes.Interface, apiExtKubeClient crd_cs.ApiextensionsV1beta1Interface) *migrator {
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
			api.ResourceNameRestic+"."+api.SchemeGroupVersion.Group,
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
		name := resourceName + "." + api.SchemeGroupVersion.Group
		if err := tprClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to remove %s TPR", name)
		}
		return nil
	}

	if err := deleteTPR(api.ResourceNameRestic); err != nil {
		return err
	}

	return nil
}

func (m *migrator) createCRDs() error {
	crds := []*crd_api.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   api.ResourceTypeRestic + "." + api.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: crd_api.CustomResourceDefinitionSpec{
				Group:   stash.GroupName,
				Version: api.SchemeGroupVersion.Version,
				Scope:   crd_api.NamespaceScoped,
				Names: crd_api.CustomResourceDefinitionNames{
					Singular:   api.ResourceNameRestic,
					Plural:     api.ResourceTypeRestic,
					Kind:       api.ResourceKindRestic,
					ShortNames: []string{"rst"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := m.apiExtKubeClient.CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = m.apiExtKubeClient.CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return apiext_util.WaitForCRDReady(m.kubeClient.CoreV1().RESTClient(), crds)
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
	crdClient := m.apiExtKubeClient.CustomResourceDefinitions()

	deleteCRD := func(resourceType string) error {
		name := resourceType + "." + api.SchemeGroupVersion.Group
		err := crdClient.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf(`Failed to delete CRD "%s""`, name)
		}
		return nil
	}

	if err := deleteCRD(api.ResourceTypeRestic); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPRs() error {
	if err := m.createTPR(api.ResourceNameRestic); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPR(resourceName string) error {
	name := resourceName + "." + api.SchemeGroupVersion.Group
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
				Name: api.SchemeGroupVersion.Version,
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

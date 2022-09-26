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

package rbac

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type Options struct {
	kubeClient              kubernetes.Interface
	owner                   *metav1.OwnerReference
	invOpts                 invokerOptions
	offshootLabels          map[string]string
	pspNames                []string
	serviceAccount          metav1.ObjectMeta
	crossNamespaceResources *crossNamespaceResources
	suffix                  string
}

type invokerOptions struct {
	metav1.ObjectMeta
	metav1.TypeMeta
}

type crossNamespaceResources struct {
	Namespace  string
	Repository string
	Secret     string
}

func NewRBACOptions(
	kubeClient kubernetes.Interface,
	inv invoker.MetadataHandler,
	repo *v1alpha1.Repository,
	index *int,
) (*Options, error) {
	invMeta := inv.GetObjectMeta()
	rbacOptions := &Options{
		kubeClient: kubeClient,
		invOpts: invokerOptions{
			ObjectMeta: invMeta,
			TypeMeta:   inv.GetTypeMeta(),
		},
		owner:          inv.GetOwnerRef(),
		offshootLabels: inv.GetLabels(),
		serviceAccount: metav1.ObjectMeta{
			Namespace: invMeta.Namespace,
		},
	}

	if repo != nil && repo.Namespace != invMeta.Namespace {
		rbacOptions.crossNamespaceResources = &crossNamespaceResources{
			Repository: repo.Name,
			Namespace:  repo.Namespace,
			Secret:     repo.Spec.Backend.StorageSecretName,
		}
	}
	rbacOptions.suffix = "0"
	if index != nil {
		rbacOptions.suffix = fmt.Sprintf("%d", *index)
	}
	return rbacOptions, nil
}

func (opt *Options) SetPSPNames(pspNames []string) {
	opt.pspNames = pspNames
}

func (opt *Options) GetServiceAccountName() string {
	return opt.serviceAccount.Name
}

func (opt *Options) SetServiceAccountName(saName string) {
	opt.serviceAccount.Name = saName
}

func (opt *Options) SetOptionsFromRuntimeSettings(runtimeSettings ofst.RuntimeSettings) {
	if runtimeSettings.Pod != nil {
		if runtimeSettings.Pod.ServiceAccountName != "" {
			opt.serviceAccount.Name = runtimeSettings.Pod.ServiceAccountName
		}

		if runtimeSettings.Pod.ServiceAccountAnnotations != nil {
			opt.serviceAccount.Annotations = runtimeSettings.Pod.ServiceAccountAnnotations
		}
	}
}

func (opt *Options) SetOwner(owner *metav1.OwnerReference) {
	opt.owner = owner
}

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RBACOptions struct {
	KubeClient              kubernetes.Interface
	Invoker                 InvokerOptions
	Owner                   *metav1.OwnerReference
	OffshootLabels          map[string]string
	PodSecurityPolicyNames  []string
	ServiceAccount          metav1.ObjectMeta
	CrossNamespaceResources *CrossNamespaceResources
	Suffix                  string
}

type InvokerOptions struct {
	metav1.ObjectMeta
	metav1.TypeMeta
}

type CrossNamespaceResources struct {
	Namespace  string
	Repository string
	Secret     string
}

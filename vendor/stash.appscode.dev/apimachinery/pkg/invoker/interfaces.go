/*
Copyright AppsCode Inc. and Contributors

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

package invoker

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type MetadataHandler interface {
	GetObjectMeta() metav1.ObjectMeta
	GetTypeMeta() metav1.TypeMeta
	GetObjectRef() (*core.ObjectReference, error)
	GetOwnerRef() *metav1.OwnerReference
	GetLabels() map[string]string
	AddFinalizer() error
	RemoveFinalizer() error
}

type ConditionHandler interface {
	HasCondition(target *v1beta1.TargetRef, conditionType string) (bool, error)
	GetCondition(target *v1beta1.TargetRef, conditionType string) (int, *kmapi.Condition, error)
	SetCondition(target *v1beta1.TargetRef, newCondition kmapi.Condition) error
	IsConditionTrue(target *v1beta1.TargetRef, conditionType string) (bool, error)
}

type RepositoryGetter interface {
	GetRepoRef() kmapi.ObjectReference
	GetRepository() (*v1alpha1.Repository, error)
}

type DriverHandler interface {
	GetDriver() v1beta1.Snapshotter
}

type TimeOutGetter interface {
	GetTimeOut() string
}

type Eventer interface {
	CreateEvent(eventType, source, reason, message string) error
}

type KubeDBIntegrator interface {
	EnsureKubeDBIntegration(appClient appcatalog_cs.Interface) error
}

type ObjectFormatter interface {
	GetHash() string
	GetObjectJSON() (string, error)
}

type Summarizer interface {
	GetSummary(target v1beta1.TargetRef, session kmapi.ObjectReference) *v1beta1.Summary
}

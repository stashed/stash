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

package apiextensions

import (
	"context"
	"sync"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SetupFn func(ctx context.Context, mgr ctrl.Manager)

var setupFns = map[schema.GroupKind]SetupFn{
	// schema.GroupKind{"compute.gcp.kubedb.com", "Firewall"}:                 firewall.Setup,
}

var (
	setupDone = map[schema.GroupKind]bool{}
	mu        sync.Mutex
)

type Reconciler struct {
	ctx context.Context
	mgr ctrl.Manager
}

func NewReconciler(ctx context.Context, mgr ctrl.Manager) *Reconciler {
	return &Reconciler{ctx: ctx, mgr: mgr}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var crd apiextensions.CustomResourceDefinition
	if err := r.mgr.GetClient().Get(ctx, req.NamespacedName, &crd); err != nil {
		log.Error(err, "unable to fetch CustomResourceDefinition")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}
	mu.Lock()
	defer mu.Unlock()
	_, found := setupDone[gk]
	if found {
		return ctrl.Result{}, nil
	}
	setup, found := setupFns[gk]
	if found {
		setup(r.ctx, r.mgr)
		setupDone[gk] = true
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextensions.CustomResourceDefinition{}).
		Complete(r)
}

func RegisterSetup(gk schema.GroupKind, fn SetupFn) {
	mu.Lock()
	defer mu.Unlock()

	setupFns[gk] = fn
}

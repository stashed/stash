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

package lib

import (
	api "go.bytebuilders.dev/audit/api/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"kmodules.xyz/client-go/discovery"
)

type BillingEventCreator struct {
	Mapper    discovery.ResourceMapper
	LicenseID string
}

func (p *BillingEventCreator) CreateEvent(obj runtime.Object) (*api.Event, error) {
	r := obj.DeepCopyObject()
	m, err := meta.Accessor(r)
	if err != nil {
		return nil, err
	}
	m.SetManagedFields(nil)

	rid, err := p.Mapper.ResourceIDForGVK(obj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, err
	}

	return &api.Event{
		Resource:   r,
		ResourceID: *rid,
		LicenseID:  p.LicenseID,
	}, nil
}

/*
Copyright The Kmodules Authors.

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
package admission

import "k8s.io/apimachinery/pkg/runtime"

// ResourceHandler can handle admission requests that happen to a
// resource.
//  * OnCreate is called when an object is created. If an error is
//      returned admission is denied. Otherwise, if an object is
//      returned, it is used to compute a patch and should be used as
//      MutatingAdmissionWebhook.
//  * OnUpdate is called when an object is updated. Note that oldObj is
//      the existing object.  If an error is  returned admission is denied.
//      Otherwise, if an object is returned, it is used to compute a patch
//      and should be used as MutatingAdmissionWebhook.
//  * OnDelete will gets the current state of object when delete request
//      is received.
type ResourceHandler interface {
	OnCreate(obj runtime.Object) (runtime.Object, error)
	OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error)
	OnDelete(obj runtime.Object) error
}

// ResourceHandlerFuncs is an adaptor to let you easily specify as many or
// as few of the notification functions as you want while still implementing
// ResourceHandler.
type ResourceHandlerFuncs struct {
	CreateFunc func(obj runtime.Object) (runtime.Object, error)
	UpdateFunc func(oldObj, newObj runtime.Object) (runtime.Object, error)
	DeleteFunc func(obj runtime.Object) error
}

// OnCreate calls CreateFunc if it's not nil.
func (r ResourceHandlerFuncs) OnCreate(obj runtime.Object) (runtime.Object, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(obj)
	}
	return nil, nil
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r ResourceHandlerFuncs) OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error) {
	if r.UpdateFunc != nil {
		return r.UpdateFunc(oldObj, newObj)
	}
	return nil, nil
}

// OnDelete calls DeleteFunc if it's not nil.
func (r ResourceHandlerFuncs) OnDelete(obj runtime.Object) error {
	if r.DeleteFunc != nil {
		return r.DeleteFunc(obj)
	}
	return nil
}

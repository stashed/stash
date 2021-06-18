// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UnstructuredEvent) DeepCopyInto(out *UnstructuredEvent) {
	*out = *in
	out.ResourceID = in.ResourceID
	if in.Resource != nil {
		in, out := &in.Resource, &out.Resource
		*out = (*in).DeepCopy()
	}
	if in.Connections != nil {
		in, out := &in.Connections, &out.Connections
		*out = make(map[schema.GroupVersionResource][]*metav1.PartialObjectMetadata, len(*in))
		for key, val := range *in {
			var outVal []*metav1.PartialObjectMetadata
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]*metav1.PartialObjectMetadata, len(*in))
				for i := range *in {
					if (*in)[i] != nil {
						in, out := &(*in)[i], &(*out)[i]
						*out = new(metav1.PartialObjectMetadata)
						(*in).DeepCopyInto(*out)
					}
				}
			}
			(*out)[key] = outVal
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UnstructuredEvent.
func (in *UnstructuredEvent) DeepCopy() *UnstructuredEvent {
	if in == nil {
		return nil
	}
	out := new(UnstructuredEvent)
	in.DeepCopyInto(out)
	return out
}

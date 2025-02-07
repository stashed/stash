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
	"context"
	"reflect"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodReader struct {
	delegate corelisters.PodLister
}

var _ client.Reader = &PodReader{}

func NewPodReader(delegate corelisters.PodLister) PodReader {
	return PodReader{delegate: delegate}
}

func (r PodReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	pod, err := r.delegate.Pods(key.Namespace).Get(key.Name)
	if err != nil {
		return err
	}
	assign(obj, pod)
	return nil
}

func (r PodReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	var o client.ListOptions
	o.ApplyOptions(opts)

	var ls labels.Selector
	if o.LabelSelector != nil {
		ls = o.LabelSelector
	} else if o.Raw != nil && o.Raw.LabelSelector != "" {
		var err error
		ls, err = labels.Parse(o.Raw.LabelSelector)
		if err != nil {
			return err
		}
	}

	pods, err := r.delegate.Pods(o.Namespace).List(ls)
	if err != nil {
		return err
	}

	podList := core.PodList{
		Items: make([]core.Pod, 0, len(pods)),
	}
	for _, pod := range pods {
		podList.Items = append(podList.Items, *pod)
	}
	assign(list, podList)

	return nil
}

type PersistentVolumeClaimReader struct {
	delegate corelisters.PersistentVolumeClaimLister
}

var _ client.Reader = &PersistentVolumeClaimReader{}

func NewPersistentVolumeClaimReader(delegate corelisters.PersistentVolumeClaimLister) PersistentVolumeClaimReader {
	return PersistentVolumeClaimReader{delegate: delegate}
}

func (r PersistentVolumeClaimReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	pod, err := r.delegate.PersistentVolumeClaims(key.Namespace).Get(key.Name)
	if err != nil {
		return err
	}
	assign(obj, pod)

	return nil
}

func (r PersistentVolumeClaimReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	var o client.ListOptions
	o.ApplyOptions(opts)

	var ls labels.Selector
	if o.LabelSelector != nil {
		ls = o.LabelSelector
	} else if o.Raw != nil && o.Raw.LabelSelector != "" {
		var err error
		ls, err = labels.Parse(o.Raw.LabelSelector)
		if err != nil {
			return err
		}
	}

	pvcs, err := r.delegate.PersistentVolumeClaims(o.Namespace).List(ls)
	if err != nil {
		return err
	}

	pvcList := core.PersistentVolumeClaimList{
		Items: make([]core.PersistentVolumeClaim, 0, len(pvcs)),
	}
	for _, pvc := range pvcs {
		pvcList.Items = append(pvcList.Items, *pvc)
	}
	assign(list, pvcList)

	return nil
}

func assign(target, src any) {
	srcValue := reflect.ValueOf(src)
	if srcValue.Kind() == reflect.Pointer {
		srcValue = srcValue.Elem()
	}
	reflect.ValueOf(target).Elem().Set(srcValue)
}

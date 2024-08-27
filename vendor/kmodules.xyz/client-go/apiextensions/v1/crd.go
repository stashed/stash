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

package v1

import (
	"context"
	"time"

	"github.com/pkg/errors"
	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

const (
	RetryTimeout = 10 * time.Minute
)

func CreateOrUpdateCustomResourceDefinition(
	ctx context.Context,
	c cs.Interface,
	name string,
	transform func(in *api.CustomResourceDefinition) *api.CustomResourceDefinition,
	opts metav1.UpdateOptions,
) (*api.CustomResourceDefinition, kutil.VerbType, error) {
	_, err := c.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating CustomResourceDefinition %s.", name)
		out, err := c.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, transform(&api.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	cur, err := TryUpdateCustomResourceDefinition(ctx, c, name, transform, opts)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return cur, kutil.VerbUpdated, nil
}

func UpdateCustomResourceDefinitionIfPresent(
	ctx context.Context,
	c cs.Interface,
	name string,
	transform func(in *api.CustomResourceDefinition) *api.CustomResourceDefinition,
	opts metav1.UpdateOptions,
) (*api.CustomResourceDefinition, kutil.VerbType, error) {
	_, err := c.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return transform(&api.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}), kutil.VerbUnchanged, nil
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	cur, err := TryUpdateCustomResourceDefinition(ctx, c, name, transform, opts)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return cur, kutil.VerbUpdated, nil
}

func TryUpdateCustomResourceDefinition(
	ctx context.Context,
	c cs.Interface,
	name string,
	transform func(*api.CustomResourceDefinition) *api.CustomResourceDefinition,
	opts metav1.UpdateOptions,
) (result *api.CustomResourceDefinition, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update CustomResourceDefinition %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update CustomResourceDefinition %s after %d attempts due to %v", name, attempt, err)
	}
	return
}

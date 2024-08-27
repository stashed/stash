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

package client

import (
	"context"
	"time"

	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#token-controller
func getServiceAccountTokenSecret(kc client.Client, sa client.ObjectKey) (*core.Secret, error) {
	var list core.SecretList
	err := kc.List(context.TODO(), &list, client.InNamespace(sa.Namespace))
	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, errors.New("token secret still haven't created yet")
	}
	for _, s := range list.Items {
		if s.Type == core.SecretTypeServiceAccountToken &&
			s.Annotations[core.ServiceAccountNameKey] == sa.Name {

			_, caFound := s.Data["ca.crt"]
			_, tokenFound := s.Data["token"]
			if caFound && tokenFound {
				return &s, nil
			}
		}
	}
	return nil, errors.New("token secret is not ready yet")
}

const (
	RetryTimeout = 10 * time.Second
)

func tryGetServiceAccountTokenSecret(kc client.Client, sa client.ObjectKey) (secret *core.Secret, err error) {
	err = wait.PollUntilContextTimeout(context.Background(), kutil.RetryInterval, RetryTimeout, true, func(ctx context.Context) (bool, error) {
		var e2 error
		secret, e2 = getServiceAccountTokenSecret(kc, sa)
		if e2 == nil {
			return true, nil
		}
		klog.V(5).Infof("trying to get token secret for service account %s", sa)
		return false, nil
	})
	return
}

func GetServiceAccountTokenSecret(kc client.Client, sa client.ObjectKey) (*core.Secret, error) {
	secret, err := tryGetServiceAccountTokenSecret(kc, sa)
	if err == nil {
		klog.V(5).Infof("secret found for ServiceAccount %s", sa)
		return secret, nil
	}

	var saObj core.ServiceAccount
	err = kc.Get(context.TODO(), sa, &saObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ServiceAccount %s", sa)
	}

	secretName := sa.Name + "-token-" + utilrand.String(6)
	secret = &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: sa.Namespace,
		},
	}
	vt, err := CreateOrPatch(context.TODO(), kc, secret, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*core.Secret)

		in.Type = core.SecretTypeServiceAccountToken
		ref := metav1.NewControllerRef(&saObj, core.SchemeGroupVersion.WithKind("ServiceAccount"))
		core_util.EnsureOwnerReference(in, ref)
		in.Annotations = meta_util.OverwriteKeys(in.Annotations, map[string]string{
			core.ServiceAccountNameKey: sa.Name,
		})

		return in
	})
	if err != nil {
		return nil, err
	}
	klog.Infof("%s Secret %s/%s", vt, secret.Namespace, secret.Name)

	return tryGetServiceAccountTokenSecret(kc, sa)
}

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

	meta_util "kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

// https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#token-controller
func getServiceAccountTokenSecret(kc kubernetes.Interface, sa types.NamespacedName) (*core.Secret, error) {
	list, err := kc.CoreV1().Secrets(sa.Namespace).List(context.TODO(), metav1.ListOptions{})
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
	retryTimeout = 10 * time.Second
)

func tryGetServiceAccountTokenSecret(kc kubernetes.Interface, sa types.NamespacedName) (secret *core.Secret, err error) {
	err = wait.PollUntilContextTimeout(context.Background(), kutil.RetryInterval, retryTimeout, true, func(ctx context.Context) (bool, error) {
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

func GetServiceAccountTokenSecret(kc kubernetes.Interface, sa types.NamespacedName) (*core.Secret, error) {
	secret, err := tryGetServiceAccountTokenSecret(kc, sa)
	if err == nil {
		klog.V(5).Infof("secret found for ServiceAccount %s", sa)
		return secret, nil
	}

	saObj, err := kc.CoreV1().ServiceAccounts(sa.Namespace).Get(context.TODO(), sa.Name, metav1.GetOptions{})
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
	secret, vt, err := CreateOrPatchSecret(context.TODO(), kc, secret.ObjectMeta, func(in *core.Secret) *core.Secret {
		in.Type = core.SecretTypeServiceAccountToken
		ref := metav1.NewControllerRef(saObj, core.SchemeGroupVersion.WithKind("ServiceAccount"))
		EnsureOwnerReference(in, ref)
		in.Annotations = meta_util.OverwriteKeys(in.Annotations, map[string]string{
			core.ServiceAccountNameKey: sa.Name,
		})

		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	klog.Infof("%s Secret %s/%s", vt, secret.Namespace, secret.Name)

	return tryGetServiceAccountTokenSecret(kc, sa)
}

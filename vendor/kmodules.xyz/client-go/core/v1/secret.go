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

package v1

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchSecret(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Secret) *core.Secret, forceSyncType ...bool) (*core.Secret, kutil.VerbType, error) {
	syncType := len(forceSyncType) == 1 && forceSyncType[0]

	cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Secret %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Secrets(meta.Namespace).Create(transform(&core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	mod := transform(cur.DeepCopy())
	if mod.Type != cur.Type && syncType {
		// secret type can't be modified once created, so we have to delete first, then recreate with correct type
		glog.Warningf("Secret %s/%s type is modified, deleting first.", meta.Namespace, meta.Name)
		err = c.CoreV1().Secrets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}

		glog.V(3).Infof("Creating Secret %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Secrets(meta.Namespace).Create(mod)
		return out, kutil.VerbCreated, err
	}
	return PatchSecretObject(c, cur, mod)
}

func PatchSecret(c kubernetes.Interface, cur *core.Secret, transform func(*core.Secret) *core.Secret) (*core.Secret, kutil.VerbType, error) {
	return PatchSecretObject(c, cur, transform(cur.DeepCopy()))
}

func PatchSecretObject(c kubernetes.Interface, cur, mod *core.Secret) (*core.Secret, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Secret{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Secret %s/%s", cur.Namespace, cur.Name)
	out, err := c.CoreV1().Secrets(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateSecret(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Secret) *core.Secret) (result *core.Secret, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Secrets(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Secret %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update Secret %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func ObfuscateSecret(in core.Secret) *core.Secret {
	data := make(map[string][]byte)
	for k := range in.Data {
		data[k] = []byte("-")
	}
	in.Data = data
	return &in
}

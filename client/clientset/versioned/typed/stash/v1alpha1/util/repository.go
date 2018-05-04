package util

import (
	"fmt"

	"github.com/appscode/kutil"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateOrPatchRepository(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(alert *api.Repository) *api.Repository) (*api.Repository, kutil.VerbType, error) {
	cur, err := c.Repositories(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Repository %s/%s.", meta.Namespace, meta.Name)
		out, err := c.Repositories(meta.Namespace).Create(transform(&api.Repository{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Repository",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchRepository(c, cur, transform)
}

func PatchRepository(c cs.StashV1alpha1Interface, cur *api.Repository, transform func(*api.Repository) *api.Repository) (*api.Repository, kutil.VerbType, error) {
	return PatchRepositoryObject(c, cur, transform(cur.DeepCopy()))
}

func PatchRepositoryObject(c cs.StashV1alpha1Interface, cur, mod *api.Repository) (*api.Repository, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Repository %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.Repositories(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateRepository(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(*api.Repository) *api.Repository) (result *api.Repository, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Repositories(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.Repositories(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Repository %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Repository %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateRepositoryStatus(c cs.StashV1alpha1Interface, cur *api.Repository, transform func(*api.RepositoryStatus) *api.RepositoryStatus, useSubresource ...bool) (*api.Repository, error) {
	if len(useSubresource) > 1 {
		return nil, errors.Errorf("invalid value passed for useSubresource: %v", useSubresource)
	}

	mod := &api.Repository{
		TypeMeta:   cur.TypeMeta,
		ObjectMeta: cur.ObjectMeta,
		Spec:       cur.Spec,
		Status:     *transform(cur.Status.DeepCopy()),
	}

	if len(useSubresource) == 1 && useSubresource[0] {
		return c.Repositories(cur.Namespace).UpdateStatus(mod)
	}

	out, _, err := PatchRepositoryObject(c, cur, mod)
	return out, err
}

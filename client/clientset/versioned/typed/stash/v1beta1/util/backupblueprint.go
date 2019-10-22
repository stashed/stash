package util

import (
	"fmt"

	api "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchBackupBlueprint(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(in *api.BackupBlueprint) *api.BackupBlueprint) (*api.BackupBlueprint, kutil.VerbType, error) {
	cur, err := c.BackupBlueprints().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating BackupBlueprint %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BackupBlueprints().Create(transform(&api.BackupBlueprint{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BackupBlueprint",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchBackupBlueprint(c, cur, transform)
}

func PatchBackupBlueprint(c cs.StashV1beta1Interface, cur *api.BackupBlueprint, transform func(*api.BackupBlueprint) *api.BackupBlueprint) (*api.BackupBlueprint, kutil.VerbType, error) {
	return PatchBackupBlueprintObject(c, cur, transform(cur.DeepCopy()))
}

func PatchBackupBlueprintObject(c cs.StashV1beta1Interface, cur, mod *api.BackupBlueprint) (*api.BackupBlueprint, kutil.VerbType, error) {
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
	glog.V(3).Infof("Patching BackupBlueprint %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BackupBlueprints().Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateBackupBlueprint(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api.BackupBlueprint) *api.BackupBlueprint) (result *api.BackupBlueprint, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BackupBlueprints().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BackupBlueprints().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update BackupBlueprint %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update BackupBlueprint %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

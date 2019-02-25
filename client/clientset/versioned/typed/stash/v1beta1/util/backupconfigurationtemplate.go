package util

import (
	"fmt"

	"github.com/appscode/kutil"
	api "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateOrPatchBackupTemplate(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(in *api.BackupConfigurationTemplate) *api.BackupConfigurationTemplate) (*api.BackupConfigurationTemplate, kutil.VerbType, error) {
	cur, err := c.BackupConfigurationTemplates().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating BackupConfigurationTemplate %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BackupConfigurationTemplates().Create(transform(&api.BackupConfigurationTemplate{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BackupConfigurationTemplate",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchBackupTemplate(c, cur, transform)
}

func PatchBackupTemplate(c cs.StashV1beta1Interface, cur *api.BackupConfigurationTemplate, transform func(*api.BackupConfigurationTemplate) *api.BackupConfigurationTemplate) (*api.BackupConfigurationTemplate, kutil.VerbType, error) {
	return PatchBackupTemplateObject(c, cur, transform(cur.DeepCopy()))
}

func PatchBackupTemplateObject(c cs.StashV1beta1Interface, cur, mod *api.BackupConfigurationTemplate) (*api.BackupConfigurationTemplate, kutil.VerbType, error) {
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
	glog.V(3).Infof("Patching BackupConfigurationTemplate %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BackupConfigurationTemplates().Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateBackupTemplate(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api.BackupConfigurationTemplate) *api.BackupConfigurationTemplate) (result *api.BackupConfigurationTemplate, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BackupConfigurationTemplates().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BackupConfigurationTemplates().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update BackupConfigurationTemplate %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update BackupConfigurationTemplate %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

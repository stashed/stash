package util

import (
	"fmt"

	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchBackupSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(bs *api.BackupSession) *api.BackupSession) (*api.BackupSession, kutil.VerbType, error) {
	cur, err := c.BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating BackupSession %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BackupSessions(meta.Namespace).Create(transform(&api.BackupSession{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BackupSession",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchBackupSession(c, cur, transform)
}

func PatchBackupSession(c cs.StashV1beta1Interface, cur *api.BackupSession, transform func(*api.BackupSession) *api.BackupSession) (*api.BackupSession, kutil.VerbType, error) {
	return PatchBackupSessionObject(c, cur, transform(cur.DeepCopy()))
}

func PatchBackupSessionObject(c cs.StashV1beta1Interface, cur, mod *api.BackupSession) (*api.BackupSession, kutil.VerbType, error) {
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
	glog.V(3).Infof("Patching BackupSession %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BackupSessions(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateBackupSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api.BackupSession) *api.BackupSession) (result *api.BackupSession, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BackupSessions(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update BackupSession %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update BackupSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func SetBackupSessionStats(c cs.StashV1beta1Interface, backupSession *api.BackupSession, backupStats []api.BackupStats, phase api.BackupSessionPhase) (*api.BackupSession, error) {
	out, err := UpdateBackupSessionStatus(c, backupSession, func(in *api.BackupSessionStatus) *api.BackupSessionStatus {
		in.Stats = backupStats
		in.Phase = phase
		return in
	}, apis.EnableStatusSubresource)
	return out, err
}

func UpdateBackupSessionStatus(
	c cs.StashV1beta1Interface,
	in *api.BackupSession,
	transform func(*api.BackupSessionStatus) *api.BackupSessionStatus,
	useSubresource ...bool,
) (result *api.BackupSession, err error) {
	if len(useSubresource) > 1 {
		return nil, errors.Errorf("invalid value passed for useSubresource: %v", useSubresource)
	}
	apply := func(x *api.BackupSession) *api.BackupSession {
		out := &api.BackupSession{
			TypeMeta:   x.TypeMeta,
			ObjectMeta: x.ObjectMeta,
			Spec:       x.Spec,
			Status:     *transform(in.Status.DeepCopy()),
		}
		return out
	}

	if len(useSubresource) == 1 && useSubresource[0] {
		attempt := 0
		cur := in.DeepCopy()
		err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
			attempt++
			var e2 error
			result, e2 = c.BackupSessions(in.Namespace).UpdateStatus(apply(cur))
			if kerr.IsConflict(e2) {
				latest, e3 := c.BackupSessions(in.Namespace).Get(in.Name, metav1.GetOptions{})
				switch {
				case e3 == nil:
					cur = latest
					return false, nil
				case kutil.IsRequestRetryable(e3):
					return false, nil
				default:
					return false, e3
				}
			} else if err != nil && !kutil.IsRequestRetryable(e2) {
				return false, e2
			}
			return e2 == nil, nil
		})

		if err != nil {
			err = fmt.Errorf("failed to update status of BackupSession %s/%s after %d attempts due to %v", in.Namespace, in.Name, attempt, err)
		}
		return
	}

	result, _, err = PatchBackupSessionObject(c, in, apply(in))
	return
}

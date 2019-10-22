package util

import (
	"fmt"

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchBackupSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(bs *api_v1beta1.BackupSession) *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
	cur, err := c.BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating BackupSession %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BackupSessions(meta.Namespace).Create(transform(&api_v1beta1.BackupSession{
			TypeMeta: metav1.TypeMeta{
				Kind:       api_v1beta1.ResourceKindBackupSession,
				APIVersion: api_v1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchBackupSession(c, cur, transform)
}

func PatchBackupSession(c cs.StashV1beta1Interface, cur *api_v1beta1.BackupSession, transform func(*api_v1beta1.BackupSession) *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
	return PatchBackupSessionObject(c, cur, transform(cur.DeepCopy()))
}

func PatchBackupSessionObject(c cs.StashV1beta1Interface, cur, mod *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
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

func TryUpdateBackupSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api_v1beta1.BackupSession) *api_v1beta1.BackupSession) (result *api_v1beta1.BackupSession, err error) {
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

func UpdateBackupSessionStatusForHost(c cs.StashV1beta1Interface, backupSession *api_v1beta1.BackupSession, hostStats api_v1beta1.HostBackupStats) (*api_v1beta1.BackupSession, error) {

	out, err := UpdateBackupSessionStatus(c, backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		// if an entry already exist for this host then update it
		for i, v := range backupSession.Status.Stats {
			if v.Hostname == hostStats.Hostname {
				in.Stats[i] = hostStats
				return in
			}
		}
		// no entry for this host. so add a new entry.
		in.Stats = append(in.Stats, hostStats)
		return in
	})
	return out, err
}

func UpdateBackupSessionStatus(
	c cs.StashV1beta1Interface,
	in *api_v1beta1.BackupSession,
	transform func(*api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus,
) (result *api_v1beta1.BackupSession, err error) {
	apply := func(x *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		out := &api_v1beta1.BackupSession{
			TypeMeta:   x.TypeMeta,
			ObjectMeta: x.ObjectMeta,
			Spec:       x.Spec,
			Status:     *transform(in.Status.DeepCopy()),
		}
		return out
	}

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

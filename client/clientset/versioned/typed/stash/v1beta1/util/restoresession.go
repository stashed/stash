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

func CreateOrPatchRestoreSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(in *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
	cur, err := c.RestoreSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating RestoreSession %s/%s.", meta.Namespace, meta.Name)
		out, err := c.RestoreSessions(meta.Namespace).Create(transform(&api_v1beta1.RestoreSession{
			TypeMeta: metav1.TypeMeta{
				Kind:       api_v1beta1.ResourceKindRestoreSession,
				APIVersion: api_v1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchRestoreSession(c, cur, transform)
}

func PatchRestoreSession(c cs.StashV1beta1Interface, cur *api_v1beta1.RestoreSession, transform func(*api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
	return PatchRestoreSessionObject(c, cur, transform(cur.DeepCopy()))
}

func PatchRestoreSessionObject(c cs.StashV1beta1Interface, cur, mod *api_v1beta1.RestoreSession) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
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
	glog.V(3).Infof("Patching RestoreSession %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.RestoreSessions(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateRestoreSession(c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession) (result *api_v1beta1.RestoreSession, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RestoreSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RestoreSessions(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update RestoreSession %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update RestoreSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateRestoreSessionStatusForHost(c cs.StashV1beta1Interface, restoreSession *api_v1beta1.RestoreSession, hostStats api_v1beta1.HostRestoreStats) (*api_v1beta1.RestoreSession, error) {
	out, err := UpdateRestoreSessionStatus(c, restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		// if an entry already exist for this host then update it
		for i, v := range restoreSession.Status.Stats {
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

func UpdateRestoreSessionStatus(
	c cs.StashV1beta1Interface,
	in *api_v1beta1.RestoreSession,
	transform func(*api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus,
) (result *api_v1beta1.RestoreSession, err error) {
	apply := func(x *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession {
		out := &api_v1beta1.RestoreSession{
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
		result, e2 = c.RestoreSessions(in.Namespace).UpdateStatus(apply(cur))
		if kerr.IsConflict(e2) {
			latest, e3 := c.RestoreSessions(in.Namespace).Get(in.Name, metav1.GetOptions{})
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
		err = fmt.Errorf("failed to update status of RestoreSession %s/%s after %d attempts due to %v", in.Namespace, in.Name, attempt, err)
	}
	return
}

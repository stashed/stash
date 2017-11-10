package util

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/wait"
)

func EnsureRecovery(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(alert *api.Recovery) *api.Recovery) (*api.Recovery, error) {
	return CreateOrPatchRecovery(c, meta, transform)
}

func CreateOrPatchRecovery(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(alert *api.Recovery) *api.Recovery) (*api.Recovery, error) {
	cur, err := c.Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Recovery %s/%s.", meta.Namespace, meta.Name)
		return c.Recoveries(meta.Namespace).Create(transform(&api.Recovery{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Recovery",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchRecovery(c, cur, transform)
}

func PatchRecovery(c cs.StashV1alpha1Interface, cur *api.Recovery, transform func(*api.Recovery) *api.Recovery) (*api.Recovery, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, err
	}

	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, modJson, curJson)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching Recovery %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	result, err := c.Recoveries(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return result, err
}

func TryPatchRecovery(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(*api.Recovery) *api.Recovery) (result *api.Recovery, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchRecovery(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Recovery %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Recovery %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateRecovery(c cs.StashV1alpha1Interface, meta metav1.ObjectMeta, transform func(*api.Recovery) *api.Recovery) (result *api.Recovery, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.Recoveries(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Recovery %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Recovery %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func SetRecoveryStatus(c cs.StashV1alpha1Interface, rec *api.Recovery, status api.RecoveryStatus) {
	_, err := PatchRecovery(c, rec, func(in *api.Recovery) *api.Recovery {
		in.Status = status
		return in
	})
	if err != nil {
		log.Errorln("Error updating recovery status:", rec.Status, "reason:", err)
	} else {
		log.Infoln("Updated recovery status:", rec.Status)
	}
}

func SetRecoveryStatusPhase(c cs.StashV1alpha1Interface, rec *api.Recovery, phase api.RecoveryPhase) {
	SetRecoveryStatus(c, rec, api.RecoveryStatus{Phase: phase})
}

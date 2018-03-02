package v1

import (
	"github.com/appscode/kutil"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchPV(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.PersistentVolume) *core.PersistentVolume) (*core.PersistentVolume, kutil.VerbType, error) {
	cur, err := c.CoreV1().PersistentVolumes().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating PersistentVolume %s.", meta.Name)
		out, err := c.CoreV1().PersistentVolumes().Create(transform(&core.PersistentVolume{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolume",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchPV(c, cur, transform)
}

func PatchPV(c kubernetes.Interface, cur *core.PersistentVolume, transform func(*core.PersistentVolume) *core.PersistentVolume) (*core.PersistentVolume, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.PersistentVolume{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching PersistentVolume %s with %s.", cur.Name, string(patch))
	out, err := c.CoreV1().PersistentVolumes().Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdatePV(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.PersistentVolume) *core.PersistentVolume) (result *core.PersistentVolume, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().PersistentVolumes().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().PersistentVolumes().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update PersistentVolume %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update PersistentVolume %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

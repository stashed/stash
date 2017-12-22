package v1beta1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchCronJob(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batch.CronJob) *batch.CronJob) (*batch.CronJob, bool, error) {
	cur, err := c.BatchV1beta1().CronJobs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating CronJob %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BatchV1beta1().CronJobs(meta.Namespace).Create(transform(&batch.CronJob{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: batch.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, true, err
	} else if err != nil {
		return nil, false, err
	}
	return PatchCronJob(c, cur, transform)
}

func PatchCronJob(c kubernetes.Interface, cur *batch.CronJob, transform func(*batch.CronJob) *batch.CronJob) (*batch.CronJob, bool, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, false, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, false, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, batch.CronJob{})
	if err != nil {
		return nil, false, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, false, nil
	}
	glog.V(3).Infof("Patching CronJob %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BatchV1beta1().CronJobs(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, true, err
}

func TryPatchCronJob(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batch.CronJob) *batch.CronJob) (result *batch.CronJob, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BatchV1beta1().CronJobs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, _, e2 = PatchCronJob(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch CronJob %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch CronJob %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateCronJob(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batch.CronJob) *batch.CronJob) (result *batch.CronJob, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BatchV1beta1().CronJobs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BatchV1beta1().CronJobs(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update CronJob %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update CronJob %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

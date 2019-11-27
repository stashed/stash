package util

import (
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	apps "kmodules.xyz/openshift/apis/apps/v1"
	cs "kmodules.xyz/openshift/client/clientset/versioned"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchDeploymentConfig(c cs.Interface, meta metav1.ObjectMeta, transform func(*apps.DeploymentConfig) *apps.DeploymentConfig) (*apps.DeploymentConfig, kutil.VerbType, error) {
	cur, err := c.AppsV1().DeploymentConfigs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating DeploymentConfig %s/%s.", meta.Namespace, meta.Name)
		out, err := c.AppsV1().DeploymentConfigs(meta.Namespace).Create(transform(&apps.DeploymentConfig{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DeploymentConfig",
				APIVersion: apps.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchDeploymentConfig(c, cur, transform)
}

func PatchDeploymentConfig(c cs.Interface, cur *apps.DeploymentConfig, transform func(*apps.DeploymentConfig) *apps.DeploymentConfig) (*apps.DeploymentConfig, kutil.VerbType, error) {
	return PatchDeploymentConfigObject(c, cur, transform(cur.DeepCopy()))
}

func PatchDeploymentConfigObject(c cs.Interface, cur, mod *apps.DeploymentConfig) (*apps.DeploymentConfig, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apps.DeploymentConfig{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching DeploymentConfig %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.AppsV1().DeploymentConfigs(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateDeploymentConfig(c cs.Interface, meta metav1.ObjectMeta, transform func(*apps.DeploymentConfig) *apps.DeploymentConfig) (result *apps.DeploymentConfig, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1().DeploymentConfigs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.AppsV1().DeploymentConfigs(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update DeploymentConfig %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update DeploymentConfig %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntilDeploymentConfigReady(c cs.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := c.AppsV1().DeploymentConfigs(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			return obj.Spec.Replicas == obj.Status.ReadyReplicas, nil
		}
		return false, nil
	})
}

func DeleteDeploymentConfig(kc kubernetes.Interface, occ cs.Interface, meta metav1.ObjectMeta) error {
	dc, err := occ.AppsV1().DeploymentConfigs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	deletePolicy := metav1.DeletePropagationForeground
	if err := occ.AppsV1().DeploymentConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil && !kerr.IsNotFound(err) {
		return err
	}
	sel := &metav1.LabelSelector{MatchLabels: dc.Spec.Selector}
	if err := core_util.WaitUntilPodDeletedBySelector(kc, dc.Namespace, sel); err != nil {
		return err
	}
	return nil
}

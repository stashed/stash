package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchNode(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Node) *core.Node) (*core.Node, error) {
	cur, err := c.CoreV1().Nodes().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Node %s/%s.", meta.Namespace, meta.Name)
		return c.CoreV1().Nodes().Create(transform(&core.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchNode(c, cur, transform)
}

func PatchNode(c kubernetes.Interface, cur *core.Node, transform func(*core.Node) *core.Node) (*core.Node, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Node{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching Node %s with %s", cur.Name, string(patch))
	return c.CoreV1().Nodes().Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchNode(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Node) *core.Node) (result *core.Node, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Nodes().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchNode(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Node %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Node %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

func TryUpdateNode(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Node) *core.Node) (result *core.Node, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Nodes().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Nodes().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Node %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Node %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

// NodeRunningAndReady returns whether a node is running.
func NodeRunningAndReady(node core.Node) (bool, error) {
	switch node.Status.Phase {
	case core.NodePending:
		return false, errors.New("node pending")
	case core.NodeTerminated:
		return false, errors.New("node terminated")
	case core.NodeRunning:
		for _, cond := range node.Status.Conditions {
			if cond.Type != core.NodeReady {
				continue
			}
			return cond.Status == core.ConditionTrue, nil
		}
		return false, errors.New("node ready condition not found")
	}
	return false, nil
}

package v1

import (
	"encoding/json"
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

func CreateOrPatchService(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Service) *core.Service) (*core.Service, kutil.VerbType, error) {
	cur, err := c.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Service %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Services(meta.Namespace).Create(transform(&core.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchService(c, cur, transform)
}

func PatchService(c kubernetes.Interface, cur *core.Service, transform func(*core.Service) *core.Service) (*core.Service, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Service{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Service %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.CoreV1().Services(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateService(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Service) *core.Service) (result *core.Service, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Services(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Service %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Service %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func MergeServicePorts(cur, desired []core.ServicePort) []core.ServicePort {
	if len(cur) == 0 {
		return desired
	}

	// ports
	curPorts := make(map[int32]core.ServicePort)
	for _, p := range cur {
		curPorts[p.Port] = p
	}
	for i, dp := range desired {
		cp, ok := curPorts[dp.Port]

		// svc port not found
		if !ok {
			continue
		}

		if dp.NodePort == 0 {
			dp.NodePort = cp.NodePort // avoid reassigning port
		}
		if dp.Protocol == "" {
			dp.Protocol = cp.Protocol
		}
		desired[i] = dp
	}
	return desired
}

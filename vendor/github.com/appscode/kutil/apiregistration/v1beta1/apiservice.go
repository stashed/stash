package v1beta1

import (
	"github.com/appscode/kutil"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	reg "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	apireg_cs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func CreateOrPatchAPIService(c apireg_cs.Interface, name string, transform func(*reg.APIService) *reg.APIService) (*reg.APIService, kutil.VerbType, error) {
	cur, err := c.ApiregistrationV1beta1().APIServices().Get(name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating APIService %s.", name)
		out, err := c.ApiregistrationV1beta1().APIServices().Create(transform(&reg.APIService{
			TypeMeta: metav1.TypeMeta{
				Kind:       "APIService",
				APIVersion: reg.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchAPIService(c, cur, transform)
}

func PatchAPIService(c apireg_cs.Interface, cur *reg.APIService, transform func(*reg.APIService) *reg.APIService) (*reg.APIService, kutil.VerbType, error) {
	return PatchAPIServiceObject(c, cur, transform(cur.DeepCopy()))
}

func PatchAPIServiceObject(c apireg_cs.Interface, cur, mod *reg.APIService) (*reg.APIService, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, reg.APIService{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching APIService %s with %s.", cur.Name, string(patch))
	out, err := c.ApiregistrationV1beta1().APIServices().Patch(cur.Name, types.StrategicMergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateAPIService(c apireg_cs.Interface, name string, transform func(*reg.APIService) *reg.APIService) (result *reg.APIService, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.ApiregistrationV1beta1().APIServices().Get(name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.ApiregistrationV1beta1().APIServices().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update APIService %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update APIService %s after %d attempts due to %v", name, attempt, err)
	}
	return
}

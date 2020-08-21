/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// applyRestoreLogic check if  RestoreSession is configured for this workload
// and perform operation accordingly
func (c *StashController) applyRestoreLogic(w *wapi.Workload, caller string) (bool, error) {
	// check if any RestoreSession exist for this workload.
	// if exist then inject init-container container
	modified, err := c.applyRestoreSessionLogic(w, caller)
	if err != nil {
		return false, err
	}
	return modified, nil
}

func (c *StashController) applyRestoreSessionLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old RestoreSession from annotations if it does exist.
	oldrs, err := util.GetAppliedRestoreSession(w.Annotations)
	if err != nil {
		return false, err
	}
	// find existing RestoreSession for this workload
	newrs, err := util.FindRestoreSession(c.restoreSessionLister, w)
	if err != nil {
		return false, err
	}
	// if RestoreSession currently exist for this workload but it is not same as old one,
	// this means RestoreSession has been newly created/updated.
	// in this case, we have to add/update init-container container accordingly.
	if newrs != nil && !util.RestoreSessionEqual(oldrs, newrs) {
		invoker, err := apis.ExtractRestoreInvokerInfo(
			c.kubeClient,
			c.stashClient,
			api_v1beta1.ResourceKindRestoreSession,
			newrs.Name,
			newrs.Namespace,
		)
		if err != nil {
			return true, err
		}
		for _, targetInfo := range invoker.TargetsInfo {
			if targetInfo.Target != nil &&
				targetInfo.Target.Ref.Kind == w.Kind &&
				targetInfo.Target.Ref.Name == w.Name {
				err = c.ensureRestoreInitContainer(w, invoker, targetInfo, caller)
				if err != nil {
					return false, c.handleInitContainerInjectionFailure(w, invoker, targetInfo.Target.Ref, err)
				}
				return true, c.handleInitContainerInjectionSuccess(w, invoker, targetInfo.Target.Ref)
			}
		}

	} else if oldrs != nil && newrs == nil {
		// there was RestoreSession before but it does not exist now.
		// this means RestoreSession has been removed.
		// in this case, we have to delete the restore init-container
		// and remove respective annotations from the workload.
		c.ensureRestoreInitContainerDeleted(w)
		// write init-container deletion failure/success event
		return true, c.handleInitContainerDeletionSuccess(w)
	}
	return false, nil
}

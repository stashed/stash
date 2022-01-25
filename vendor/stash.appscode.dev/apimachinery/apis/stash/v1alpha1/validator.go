/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"
	"strings"
)

func (r Repository) IsValid() error {
	if r.Spec.WipeOut {
		if r.Spec.Backend.Local != nil {
			return fmt.Errorf("wipe out operation is not supported for local backend")
		} else if r.Spec.Backend.B2 != nil {
			return fmt.Errorf("wipe out operation is not supported for B2 backend")
		}
	}

	if r.Spec.Backend.Local != nil && r.Spec.Backend.Local.MountPath != "" {
		parts := strings.Split(r.Spec.Backend.Local.MountPath, "/")
		if len(parts) >= 2 && parts[1] == "stash" {
			return fmt.Errorf("\n\t" +
				"Error: Invalid `mountPath` specification for local backend.\n\t" +
				"Reason: We have put `stash` binary  in the root directory. Hence, you can not use `/stash` or `/stash/*` as `mountPath` \n\t" +
				"Hints: Use `/stash-backup` or anything else except the forbidden ones as `mountPath`.")
		}
	}
	return nil
}

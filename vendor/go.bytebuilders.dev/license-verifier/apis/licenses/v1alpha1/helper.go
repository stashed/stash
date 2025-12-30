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

func (l License) DisableAnalytics() bool {
	return len(l.FeatureFlags) > 0 && l.FeatureFlags[FeatureDisableAnalytics] == "true"
}

func (l License) EnableClientBilling() bool {
	return len(l.FeatureFlags) > 0 && l.FeatureFlags[FeatureEnableClientBilling] == "true"
}

func (l License) ActivationMode() ActivationMode {
	if l.FeatureFlags[FeatureActivationMode] == string(ActivationModeCertification) {
		return ActivationModeCertification
	}
	return ActivationModeFull
}

func (i *License) Less(j *License) bool {
	if i == nil {
		return true
	} else if j == nil {
		return false
	}

	iRank := rankTier(i.TierName)
	jRank := rankTier(j.TierName)
	if iRank != jRank {
		return iRank < jRank
	}

	if i.NotBefore == nil {
		return true
	} else if j.NotBefore == nil {
		return false
	}
	if !i.NotBefore.Equal(j.NotBefore) {
		return i.NotBefore.Before(j.NotBefore)
	}

	if i.NotAfter == nil {
		return true
	} else if j.NotAfter == nil {
		return false
	}
	return i.NotAfter.Before(j.NotAfter)
}

func rankTier(t string) int {
	// prefer enterprise licenses in a min priority queue
	switch t {
	case "enterprise":
		return 0
	case "":
		return 2
	default:
		return 1
	}
}

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

package meta

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func LabelsForLabelSelector(sel *metav1.LabelSelector) (map[string]string, bool) {
	if sel != nil {
		if len(sel.MatchExpressions) > 0 {
			expr := sel.MatchExpressions[0]
			switch expr.Operator {
			case metav1.LabelSelectorOpIn:
				return map[string]string{
					expr.Key: expr.Values[0],
				}, false
			case metav1.LabelSelectorOpNotIn:
				return map[string]string{
					expr.Key: "not-" + expr.Values[0],
				}, false
			case metav1.LabelSelectorOpExists:
				return map[string]string{
					expr.Key: "",
				}, false
			case metav1.LabelSelectorOpDoesNotExist:
				return make(map[string]string), false
			}
		} else {
			return sel.MatchLabels, true
		}
	}
	return make(map[string]string), true
}

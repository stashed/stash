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

package v1beta1

import "fmt"

// TODO: complete
func (r BackupSession) IsValid() error {
	return nil
}

// TODO: complete
func (r RestoreSession) IsValid() error {
	// ========== spec.Rules validation================
	// We must ensure following:
	// 1. There is at most one rule with empty targetHosts field.
	// 2. No two rules with non-emtpy targetHosts matches for a host.
	// 3. If snapshot field is specified in a rule then paths is not specified.

	// ensure the there is at most one rule with source
	var ruleIdx []int
	for i, rule := range r.Spec.Rules {
		if len(rule.TargetHosts) == 0 {
			ruleIdx = append(ruleIdx, i)
		}
	}
	if len(ruleIdx) > 1 {
		return fmt.Errorf("\n\t"+
			"Error: Invalid RestoreSession specification.\n\t"+
			"Reason: %s.\n\t"+
			"Hints: There can be at most one rule with empty targetHosts", multipleRuleWithEmptyTargetHostError(ruleIdx))
	}

	// ensure that no two rules with non-emtpy targetHosts matches for a host
	res := make(map[string]int)
	for i, rule := range r.Spec.Rules {
		for _, host := range rule.TargetHosts {
			v, ok := res[host]
			if ok {
				return fmt.Errorf("\n\t"+
					"Error: Invalid RestoreSession specification.\n\t"+
					"Reason: Multiple rules (rule[%d] and rule[%d]) match for host %q.\n\t"+
					"Hints: There could be only one matching rule for a host", v, i, host)
			} else {
				res[host] = i
			}
		}
	}

	// ensure that path is not specified in a rule if snapshot field is specified
	for i, rule := range r.Spec.Rules {
		if len(rule.Snapshots) != 0 && len(rule.Paths) != 0 {
			return fmt.Errorf("\n\t"+
				"Error: Invalid RestoreSession specification.\n\t"+
				"Reason: Both 'snapshots' and 'paths' fileds are specified in rule[%d].\n\t"+
				"Hints: A snpashot contains backup data of only one directory. So, you can't specify 'paths' if you specify snapshot field", i)
		}
	}
	return nil
}

func multipleRuleWithEmptyTargetHostError(ruleIndexes []int) string {
	ids := ""
	for i, idx := range ruleIndexes {
		ids += fmt.Sprintf("rule[%d]", idx)
		if i < len(ruleIndexes)-1 {
			ids += ", "
		}
	}
	return fmt.Sprintf("%d rules found with empty targetHosts (Rules: %s)", len(ruleIndexes), ids)
}

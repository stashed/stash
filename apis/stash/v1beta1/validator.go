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
	// 2. No two rule with non-emtpy targetHosts matches for a host.
	// 3. If snapshot field is specified in a rule then paths is not specified.

	// ensure the there is at most one rule with source
	var ruleIndexes []int
	for i, rule := range r.Spec.Rules {
		if len(rule.TargetHosts) == 0 {
			ruleIndexes = append(ruleIndexes, i)
		}
	}
	if len(ruleIndexes) > 1 {
		return fmt.Errorf("\n\t"+
			"Error: Invalid RestoreSession specification.\n\t"+
			"Reason: %s.\n\t"+
			"Hints: There could be at most one rule with empty targetHosts.", multipleRuleWithEmptyTargetHostError(ruleIndexes))
	}

	// ensure that no two rules with non-emtpy targetHosts matches for a host
	res := make(map[string]int, 0)
	for i, rule := range r.Spec.Rules {
		if len(rule.TargetHosts) != 0 {
			for _, host := range rule.TargetHosts {
				v, ok := res[host]
				if ok {
					return fmt.Errorf("\n\t"+
						"Error: Invalid RestoreSession specification.\n\t"+
						"Reason: Multiple rules (rule[%d] and rule[%d]) matches for host %q.\n\t"+
						"Hints: There could be only one matching rule for a host.", v, i, host)
				} else {
					res[host] = i
				}
			}
		}
	}

	// ensure that path is not specified in a rule if snapshot field is specified
	for i, rule := range r.Spec.Rules {
		if len(rule.Snapshots) != 0 && len(rule.Paths) != 0 {
			return fmt.Errorf("\n\t"+
				"Error: Invalid RestoreSession specification.\n\t"+
				"Reason: Both 'snapshots' and 'paths' fileds are specified in rule[%d].\n\t"+
				"Hints: A snpashot contains backup data of only one directory. So, you don't have to specify 'paths' if you specify snapshot field.", i)
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

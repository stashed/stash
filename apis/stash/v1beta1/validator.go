package v1beta1

import (
	"fmt"

	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

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
			"Hints: There can be at most one rule with empty targetHosts.", multipleRuleWithEmptyTargetHostError(ruleIdx))
	}

	// ensure that no two rules with non-emtpy targetHosts matches for a host
	res := make(map[string]int, 0)
	for i, rule := range r.Spec.Rules {
		for _, host := range rule.TargetHosts {
			v, ok := res[host]
			if ok {
				return fmt.Errorf("\n\t"+
					"Error: Invalid RestoreSession specification.\n\t"+
					"Reason: Multiple rules (rule[%d] and rule[%d]) match for host %q.\n\t"+
					"Hints: There could be only one matching rule for a host.", v, i, host)
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
				"Hints: A snpashot contains backup data of only one directory. So, you can't specify 'paths' if you specify snapshot field.", i)
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

func (b BackupBlueprint) IsValid() error {
	// We must ensure the following:
	// 1. Spec.schedule
	// 2. Spec.Backend.StorageSecretName
	// 3.
	if b.Spec.Schedule == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: BackupConfiguration Schedule is not specified\n\t" +
			"Hints: Schedule is a cron expression like \"* * * * *\"")
	}
	if b.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: Repository secret is not specified\n\t")
	}
	return nil

}

func (b BackupConfiguration) IsValid(kubeClient kubernetes.Interface, list *v1alpha1.ResticList) error {
	if b.Spec.Schedule == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: BackupConfiguration Schedule is not specified\n\t" +
			"Hints: Schedule is a cron expression like \"* * * * *\"")
	}
	// If the target(workload) is already invoked by Restic then
	// BackupConfiguration will not be created.
	fmt.Println("hello")
	if b.Spec.Target != nil {
		var label map[string]string
		switch b.Spec.Target.Ref.Kind {
		case apis.KindDeployment:
			workload, err := kubeClient.AppsV1().Deployments(b.Namespace).Get(b.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				} else {
					return err
				}
			}
			label = workload.Labels
		case apis.KindStatefulSet:
			workload, err := kubeClient.AppsV1().StatefulSets(b.Namespace).Get(b.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				} else {
					return err
				}
			}
			label = workload.Labels
		case apis.KindDaemonSet:
			workload, err := kubeClient.AppsV1().DaemonSets(b.Namespace).Get(b.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				} else {
					return err
				}
			}
			label = workload.Labels
		case apis.KindReplicationController:
			workload, err := kubeClient.CoreV1().ReplicationControllers(b.Namespace).Get(b.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				}
			}
			label = workload.Labels
		case apis.KindReplicaSet:
			workload, err := kubeClient.AppsV1().ReplicaSets(b.Namespace).Get(b.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				}
			}
			label = workload.Labels
		}
		for _, restic := range list.Items {
			selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
			if err != nil {
				return nil
			}
			if selector.Matches(labels.Set(label)) {
				return fmt.Errorf("\n\t" +
					"Error:  Workload is not available for BackupConfiguration to backup\n\t" +
					"Reason: Workload already invoked by Restic\n\t")
			}
		}
	}

	return nil
}

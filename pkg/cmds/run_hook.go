/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"context"
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashHooks "stash.appscode.dev/apimachinery/pkg/hooks"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type hookOptions struct {
	masterURL         string
	kubeConfigPath    string
	namespace         string
	hookType          string
	backupSessionName string
	targetRef         v1beta1.TargetRef
	invokerKind       string
	invokerName       string
	hostname          string
	config            *rest.Config
	kubeClient        kubernetes.Interface
	stashClient       cs.Interface
	appClient         appcatalog_cs.Interface
	metricOpts        metrics.MetricsOptions
	outputDir         string
}

func NewCmdRunHook() *cobra.Command {
	opt := hookOptions{
		masterURL:      "",
		kubeConfigPath: "",
		namespace:      meta.PodNamespace(),
		hostname:       apis.DefaultHost,
	}

	cmd := &cobra.Command{
		Use:               "run-hook",
		Short:             "Execute Backup or Restore Hooks",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(opt.masterURL, opt.kubeConfigPath)
			if err != nil {
				klog.Fatalf("Could not get Kubernetes config: %s", err)
				return err
			}

			opt.config = config
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.appClient = appcatalog_cs.NewForConfigOrDie(config)
			opt.metricOpts.JobName = fmt.Sprintf("%s-%s-%s", strings.ToLower(opt.invokerKind), opt.namespace, opt.invokerName)

			err = opt.executeHook()
			if err != nil {
				klog.Infof("Failed to execute %s hook. Reason: %v", opt.hookType, err)
				if opt.shouldFailContainer() {
					return err
				}
				return nil
			}
			klog.Infof("Successfully executed %q hook.", opt.hookType)
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.masterURL, "master", opt.masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.kubeConfigPath, "kubeconfig", opt.kubeConfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.backupSessionName, "backupsession", opt.backupSessionName, "Name of the respective BackupSession object")
	cmd.Flags().StringVar(&opt.invokerKind, "invoker-kind", opt.invokerKind, "Type of the backup invoker")
	cmd.Flags().StringVar(&opt.invokerName, "invoker-name", opt.invokerName, "Name of the respective backup invoker")
	cmd.Flags().StringVar(&opt.targetRef.Name, "target-name", opt.targetRef.Name, "Name of the Target")
	cmd.Flags().StringVar(&opt.targetRef.Namespace, "target-namespace", opt.targetRef.Namespace, "Namespace of the Target")
	cmd.Flags().StringVar(&opt.targetRef.Kind, "target-kind", opt.targetRef.Name, "Kind of the Target")
	cmd.Flags().StringVar(&opt.hookType, "hook-type", opt.hookType, "Type of hook to execute")
	cmd.Flags().StringVar(&opt.hostname, "hostname", opt.hostname, "Name of the host that is being backed up or restored")
	cmd.Flags().BoolVar(&opt.metricOpts.Enabled, "metrics-enabled", opt.metricOpts.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metricOpts.PushgatewayURL, "metrics-pushgateway-url", opt.metricOpts.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringSliceVar(&opt.metricOpts.Labels, "metrics-labels", opt.metricOpts.Labels, "Labels to apply in exported metrics")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")
	return cmd
}

func (opt *hookOptions) executeHook() error {
	if opt.backupSessionName != "" {
		return opt.executeBackupHook()
	}
	return opt.executeRestoreHook()
}

func (opt *hookOptions) executeBackupHook() error {
	inv, err := invoker.NewBackupInvoker(opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
	if err != nil {
		return err
	}

	targetInfo := opt.getBackupTargetInfo(inv)
	if targetInfo == nil {
		return fmt.Errorf("backup target %s/%s did not matched with any target of the %s %s/%s",
			opt.targetRef.Kind,
			opt.targetRef.Name,
			opt.invokerKind,
			opt.namespace,
			opt.invokerName,
		)
	}

	hookExecutor := stashHooks.BackupHookExecutor{
		Config:      opt.config,
		StashClient: opt.stashClient,
		Invoker:     inv,
		Target:      targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: opt.namespace,
		},
	}
	if opt.hookType == apis.PreBackupHook {
		hookExecutor.Hook = targetInfo.Hooks.PreBackup
		hookExecutor.HookType = apis.PreBackupHook
	} else {
		hookExecutor.Hook = targetInfo.Hooks.PostBackup
		hookExecutor.HookType = apis.PostBackupHook
	}
	hookExecutor.ExecutorPod.Name, err = opt.getHookExecutorPodName(targetInfo.Target.Ref)
	if err != nil {
		return err
	}

	hookExecutor.BackupSession, err = opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(context.TODO(), opt.backupSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return hookExecutor.Execute()
}

func (opt *hookOptions) getBackupTargetInfo(inv invoker.BackupInvoker) *invoker.BackupTargetInfo {
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetRef.Kind, opt.targetRef.Name, opt.targetRef.Namespace) {
			return &targetInfo
		}
	}
	return nil
}

func (opt *hookOptions) executeRestoreHook() error {
	inv, err := invoker.NewRestoreInvoker(opt.kubeClient, opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
	if err != nil {
		return err
	}

	targetInfo := opt.getRestoreTargetInfo(inv)
	if targetInfo == nil {
		return fmt.Errorf("restore target %s/%s did not matched with any target of the %s %s/%s",
			opt.targetRef.Kind,
			opt.targetRef.Name,
			opt.invokerKind,
			opt.namespace,
			opt.invokerName,
		)
	}

	hookExecutor := stashHooks.RestoreHookExecutor{
		Config:  opt.config,
		Invoker: inv,
		Target:  targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: opt.namespace,
		},
	}

	if opt.hookType == apis.PreRestoreHook {
		hookExecutor.Hook = targetInfo.Hooks.PreRestore
		hookExecutor.HookType = apis.PreRestoreHook
	} else {
		hookExecutor.Hook = targetInfo.Hooks.PostRestore
		hookExecutor.HookType = apis.PostRestoreHook
	}
	hookExecutor.ExecutorPod.Name, err = opt.getHookExecutorPodName(targetInfo.Target.Ref)
	if err != nil {
		return err
	}
	return hookExecutor.Execute()
}

func (opt *hookOptions) getRestoreTargetInfo(inv invoker.RestoreInvoker) *invoker.RestoreTargetInfo {
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetRef.Kind, opt.targetRef.Name, opt.targetRef.Namespace) {
			return &targetInfo
		}
	}
	return nil
}

func (opt *hookOptions) getHookExecutorPodName(targetRef v1beta1.TargetRef) (string, error) {
	switch targetRef.Kind {
	case apis.KindAppBinding:
		// For AppBinding, we will execute the hooks in the respective app pod
		return opt.getAppPodName(targetRef.Name)
	default:
		// For other types of target, hook will be executed where this process is running.
		return meta.PodName(), nil
	}
}

func (opt *hookOptions) getAppPodName(appbindingName string) (string, error) {
	appbinding, err := opt.appClient.AppcatalogV1alpha1().AppBindings(opt.namespace).Get(context.TODO(), appbindingName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// AppBinding should have a Service in ClientConfig field. This service selects the app pod. We will execute the hooks in the app pod.
	if appbinding.Spec.ClientConfig.Service != nil {
		// there should be an endpoint with same name as the service which contains the name of the selected pods.
		endPoint, err := opt.kubeClient.CoreV1().Endpoints(opt.namespace).Get(context.TODO(), appbinding.Spec.ClientConfig.Service.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		for _, subSets := range endPoint.Subsets {
			// get pod from the ready addresses
			for _, readyAddrs := range subSets.Addresses {
				if readyAddrs.TargetRef != nil && readyAddrs.TargetRef.Kind == apis.KindPod {
					return readyAddrs.TargetRef.Name, nil
				}
			}
			// no pod found in ready addresses. now try in not ready addresses.
			for _, notReadyAddrs := range subSets.NotReadyAddresses {
				if notReadyAddrs.TargetRef != nil && notReadyAddrs.TargetRef.Kind == apis.KindPod {
					return notReadyAddrs.TargetRef.Name, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no pod found for AppBinding %s/%s", opt.namespace, appbindingName)
}

func (opt *hookOptions) shouldFailContainer() bool {
	return opt.hookType == apis.PreBackupHook || opt.hookType == apis.PreRestoreHook
}

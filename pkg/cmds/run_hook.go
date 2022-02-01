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
	"os"
	"path/filepath"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type hookOptions struct {
	masterURL         string
	kubeConfigPath    string
	namespace         string
	hookType          string
	backupSessionName string
	targetKind        string
	targetName        string
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
		namespace:      meta.Namespace(),
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
				// For preBackup or preRestore hook failure, we will fail the container so that the task does to proceed to next step.
				// We will also update the BackupSession/RestoreSession status as the update-status Function will not execute.
				if opt.hookType == apis.PreBackupHook || opt.hookType == apis.PreRestoreHook {
					return opt.handlePreTaskHookFailure(err)
				}
				// For other postBackup or postRestore hook failure, we will simply write the failure output into the output directory.
				// The update-status Function will update the status of the BackupSession/RestoreSession
				return opt.handlePostTaskHookFailure(err)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&opt.masterURL, "master", opt.masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.kubeConfigPath, "kubeconfig", opt.kubeConfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.backupSessionName, "backupsession", opt.backupSessionName, "Name of the respective BackupSession object")
	cmd.Flags().StringVar(&opt.invokerKind, "invoker-kind", opt.invokerKind, "Type of the backup invoker")
	cmd.Flags().StringVar(&opt.invokerName, "invoker-name", opt.invokerName, "Name of the respective backup invoker")
	cmd.Flags().StringVar(&opt.targetName, "target-name", opt.targetName, "Name of the Target")
	cmd.Flags().StringVar(&opt.targetKind, "target-kind", opt.targetName, "Kind of the Target")
	cmd.Flags().StringVar(&opt.hookType, "hook-type", opt.hookType, "Type of hook to execute")
	cmd.Flags().StringVar(&opt.hostname, "hostname", opt.hostname, "Name of the host that is being backed up or restored")
	cmd.Flags().BoolVar(&opt.metricOpts.Enabled, "metrics-enabled", opt.metricOpts.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metricOpts.PushgatewayURL, "metrics-pushgateway-url", opt.metricOpts.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringSliceVar(&opt.metricOpts.Labels, "metrics-labels", opt.metricOpts.Labels, "Labels to apply in exported metrics")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")
	return cmd
}

func (opt *hookOptions) executeHook() error {
	var hook interface{}
	var executorPodName string

	if opt.backupSessionName != "" {
		// For backup hooks, BackupSession name will be provided. We will read the hooks from the underlying backup inv.
		inv, err := invoker.NewBackupInvoker(opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
		if err != nil {
			return err
		}
		// We need to extract the hook only for the current target
		for _, targetInfo := range inv.GetTargetInfo() {
			if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {
				hook = targetInfo.Hooks
				executorPodName, err = opt.getHookExecutorPodName(targetInfo.Target.Ref)
				if err != nil {
					return err
				}
				break
			}
		}
	} else {
		// backupSessionName flag name was not provided, it means it is restore hook.
		inv, err := invoker.NewRestoreInvoker(opt.kubeClient, opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
		if err != nil {
			return err
		}
		for _, targetInfo := range inv.GetTargetInfo() {
			if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {
				hook = targetInfo.Hooks
				executorPodName, err = opt.getHookExecutorPodName(targetInfo.Target.Ref)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	// Execute the hooks
	return util.ExecuteHook(opt.config, hook, opt.hookType, executorPodName, opt.namespace)
}

func (opt *hookOptions) getHookExecutorPodName(targetRef v1beta1.TargetRef) (string, error) {
	switch targetRef.Kind {
	case apis.KindAppBinding:
		// For AppBinding, we will execute the hooks in the respective app pod
		return opt.getAppPodName(targetRef.Name)
	default:
		// For other types of target, hook will be executed where this process is running.
		return os.Getenv(apis.KeyPodName), nil
	}
}

func (opt *hookOptions) getAppPodName(appbindingName string) (string, error) {
	// get the AppBinding
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

func (opt *hookOptions) handlePreTaskHookFailure(hookErr error) error {
	statusOpt := status.UpdateStatusOptions{
		Config:      opt.config,
		KubeClient:  opt.kubeClient,
		StashClient: opt.stashClient,
		Namespace:   opt.namespace,
		Metrics:     opt.metricOpts,
		TargetRef: v1beta1.TargetRef{
			Kind: opt.targetKind,
			Name: opt.targetName,
		},
	}
	if opt.hookType == apis.PreBackupHook {
		backupOutput := &restic.BackupOutput{
			BackupTargetStatus: v1beta1.BackupTargetStatus{
				Ref: statusOpt.TargetRef,
				Stats: []v1beta1.HostBackupStats{
					{
						Hostname: opt.hostname,
						Phase:    v1beta1.HostBackupFailed,
						Error:    hookErr.Error(),
					},
				},
			},
		}
		statusOpt.BackupSession = opt.backupSessionName
		// Extract invoker information
		inv, err := invoker.NewBackupInvoker(opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
		if err != nil {
			return err
		}
		for _, targetInfo := range inv.GetTargetInfo() {
			if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {
				err := statusOpt.UpdatePostBackupStatus(backupOutput, inv, targetInfo)
				if err != nil {
					hookErr = errors.NewAggregate([]error{hookErr, err})
				}
			}
		}
	} else {
		// otherwise it is postRestore hook
		restoreOutput := &restic.RestoreOutput{
			RestoreTargetStatus: v1beta1.RestoreMemberStatus{
				Ref: statusOpt.TargetRef,
				Stats: []v1beta1.HostRestoreStats{
					{
						Hostname: opt.hostname,
						Phase:    v1beta1.HostRestoreFailed,
						Error:    hookErr.Error(),
					},
				},
			},
		}
		inv, err := invoker.NewRestoreInvoker(opt.kubeClient, opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
		if err != nil {
			return err
		}

		for _, targetInfo := range inv.GetTargetInfo() {
			if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {
				err = statusOpt.UpdatePostRestoreStatus(restoreOutput, inv, targetInfo)
				if err != nil {
					hookErr = errors.NewAggregate([]error{hookErr, err})
				}
			}
		}
	}
	// return error so that the container fail
	return hookErr
}

func (opt *hookOptions) handlePostTaskHookFailure(hookErr error) error {
	if opt.hookType == apis.PostBackupHook {
		backupOutput := &restic.BackupOutput{
			BackupTargetStatus: v1beta1.BackupTargetStatus{
				Ref: v1beta1.TargetRef{
					Kind: opt.targetKind,
					Name: opt.targetName,
				},
				Stats: []v1beta1.HostBackupStats{
					{
						Hostname: opt.hostname,
						Phase:    v1beta1.HostBackupFailed,
						Error:    hookErr.Error(),
					},
				},
			},
		}

		err := backupOutput.WriteOutput(filepath.Join(opt.outputDir, restic.DefaultOutputFileName))
		if err != nil {
			// failed to write output file. we should fail the container. hence, we are returning the error
			return errors.NewAggregate([]error{hookErr, err})
		}
	} else { // otherwise it is postRestore hook
		restoreOutput := &restic.RestoreOutput{
			RestoreTargetStatus: v1beta1.RestoreMemberStatus{
				Ref: v1beta1.TargetRef{
					Kind: opt.targetKind,
					Name: opt.targetName,
				},
				Stats: []v1beta1.HostRestoreStats{
					{
						Hostname: opt.hostname,
						Phase:    v1beta1.HostRestoreFailed,
						Error:    hookErr.Error(),
					},
				},
			},
		}
		err := restoreOutput.WriteOutput(filepath.Join(opt.outputDir, restic.DefaultOutputFileName))
		if err != nil {
			// failed to write output file. we should fail the container. hence, are returning the error
			return errors.NewAggregate([]error{hookErr, err})
		}
	}
	// don't return error. we don't want to fail this container. update-status Function will execute after it.
	// update-status Function will take care of updating BackupSession/RestoreSession status
	return nil
}

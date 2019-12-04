/*
Copyright The Stash Authors.

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

package cmds

import (
	"fmt"
	"os"
	"path/filepath"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type hookOptions struct {
	masterURL          string
	kubeConfigPath     string
	namespace          string
	hookType           string
	backupSessionName  string
	restoreSessionName string
	hostname           string
	config             *rest.Config
	kubeClient         kubernetes.Interface
	stashClient        cs.Interface
	appClient          appcatalog_cs.Interface
	metricOpts         restic.MetricsOptions
	outputDir          string
}

func NewCmdRunHook() *cobra.Command {
	opt := hookOptions{
		masterURL:      "",
		kubeConfigPath: "",
		namespace:      meta.Namespace(),
		hostname:       util.DefaultHost,
	}

	cmd := &cobra.Command{
		Use:               "run-hook",
		Short:             "Execute Backup or Restore Hooks",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(opt.masterURL, opt.kubeConfigPath)
			if err != nil {
				glog.Fatalf("Could not get Kubernetes config: %s", err)
				return err
			}

			opt.config = config
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.appClient = appcatalog_cs.NewForConfigOrDie(config)

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
	cmd.Flags().StringVar(&opt.restoreSessionName, "restoresession", opt.restoreSessionName, "Name of the respective RestoreSession")
	cmd.Flags().StringVar(&opt.hookType, "hook-type", opt.hookType, "Type of hook to execute")
	cmd.Flags().StringVar(&opt.hostname, "hostname", opt.hostname, "Name of the host that is being backed up or restored")
	cmd.Flags().BoolVar(&opt.metricOpts.Enabled, "metrics-enabled", opt.metricOpts.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metricOpts.PushgatewayURL, "metrics-pushgateway-url", opt.metricOpts.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringSliceVar(&opt.metricOpts.Labels, "metrics-labels", opt.metricOpts.Labels, "Labels to apply in exported metrics")
	cmd.Flags().StringVar(&opt.metricOpts.JobName, "prom-job-name", StashDefaultMetricJob, "Metrics job name")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")
	return cmd
}

func (opt *hookOptions) executeHook() error {
	var backupConfig *v1beta1.BackupConfiguration
	var restoreSession *v1beta1.RestoreSession
	var err error

	// For backup hooks, BackupSession name will be provided. We will read the hooks from the underlying BackupConfiguration.
	if opt.backupSessionName != "" {
		backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(opt.backupSessionName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if backupSession.Spec.Invoker.Kind != v1beta1.ResourceKindBackupConfiguration {
			return fmt.Errorf("backup hook for invoker kind: %s is not supported yet", backupSession.Spec.Invoker.Kind)
		}
		backupConfig, err = opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	// For restore hooks, RestoreSession name will be provided. We will read the hooks from the RestoreSession.
	if opt.restoreSessionName != "" {
		restoreSession, err = opt.stashClient.StashV1beta1().RestoreSessions(opt.namespace).Get(opt.restoreSessionName, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	// Extract the hooks from the BackupConfiguration or RestoreSession
	hook, err := opt.getHook(backupConfig, restoreSession)
	if err != nil {
		return err
	}

	// Now, determine the pod where the hook will execute
	podName, err := opt.getPodName(backupConfig, restoreSession)
	if err != nil {
		return err
	}
	// Execute the hooks
	return util.ExecuteHook(opt.config, hook, opt.hookType, podName, opt.namespace)
}

func (opt *hookOptions) getHook(backupConfig *v1beta1.BackupConfiguration, restoreSession *v1beta1.RestoreSession) (interface{}, error) {
	switch opt.hookType {
	case apis.PreBackupHook:
		if backupConfig != nil && backupConfig.Spec.Hooks != nil && backupConfig.Spec.Hooks.PreBackup != nil {
			return backupConfig.Spec.Hooks, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in BackupConfiguration %s/%s", opt.hookType, opt.namespace, opt.backupSessionName)
		}
	case apis.PostBackupHook:
		if backupConfig != nil && backupConfig.Spec.Hooks != nil && backupConfig.Spec.Hooks.PostBackup != nil {
			return backupConfig.Spec.Hooks, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in BackupConfiguration %s/%s", opt.hookType, opt.namespace, opt.backupSessionName)
		}
	case apis.PreRestoreHook:
		if restoreSession != nil && restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PreRestore != nil {
			return restoreSession.Spec.Hooks, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in RestoreSession %s/%s", opt.hookType, opt.namespace, opt.restoreSessionName)
		}
	case apis.PostRestoreHook:
		if restoreSession != nil && restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PostRestore != nil {
			return restoreSession.Spec.Hooks, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in RestoreSession %s/%s", opt.hookType, opt.namespace, opt.restoreSessionName)
		}
	default:
		return nil, fmt.Errorf("unknown hook type: %s", opt.hookType)
	}
}

func (opt *hookOptions) getPodName(backupConfig *v1beta1.BackupConfiguration, restoreSession *v1beta1.RestoreSession) (string, error) {
	var targetRef v1beta1.TargetRef
	// only one of backupConfig or restoreSession will be not nil
	if backupConfig != nil && backupConfig.Spec.Target != nil {
		targetRef = backupConfig.Spec.Target.Ref
	} else if restoreSession != nil && restoreSession.Spec.Target != nil {
		targetRef = restoreSession.Spec.Target.Ref
	} else {
		return "", fmt.Errorf("invalid target. target can't be nil for executing hook in Function-Task model")
	}

	switch targetRef.Kind {
	case apis.KindAppBinding:
		// For AppBinding, we will execute the hooks in the respective app pod
		return opt.getAppPodName(targetRef.Name)
	default:
		return os.Getenv(util.KeyPodName), nil
	}
}

func (opt *hookOptions) getAppPodName(appbindingName string) (string, error) {
	// get the AppBinding
	appbinding, err := opt.appClient.AppcatalogV1alpha1().AppBindings(opt.namespace).Get(appbindingName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// AppBinding should have a Service in ClientConfig field. This service selects the app pod. We will execute the hooks in the app pod.
	if appbinding.Spec.ClientConfig.Service != nil {
		// there should be an endpoint with same name as the service which contains the name of the selected pods.
		endPoint, err := opt.kubeClient.CoreV1().Endpoints(opt.namespace).Get(appbinding.Spec.ClientConfig.Service.Name, metav1.GetOptions{})
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
	}
	if opt.hookType == apis.PreBackupHook {
		backupOutput := &restic.BackupOutput{
			HostBackupStats: []v1beta1.HostBackupStats{
				{
					Hostname: opt.hostname,
					Phase:    v1beta1.HostBackupFailed,
					Error:    hookErr.Error(),
				},
			},
		}
		statusOpt.BackupSession = opt.backupSessionName

		err := statusOpt.UpdatePostBackupStatus(backupOutput)
		if err != nil {
			hookErr = errors.NewAggregate([]error{hookErr, err})
		}
	} else { // otherwise it is postRestore hook
		restoreOutput := &restic.RestoreOutput{
			HostRestoreStats: []v1beta1.HostRestoreStats{
				{
					Hostname: opt.hostname,
					Phase:    v1beta1.HostRestoreFailed,
					Error:    hookErr.Error(),
				},
			},
		}
		statusOpt.RestoreSession = opt.restoreSessionName

		err := statusOpt.UpdatePostRestoreStatus(restoreOutput)
		if err != nil {
			hookErr = errors.NewAggregate([]error{hookErr, err})
		}
	}
	// return error so that the container fail
	return hookErr
}

func (opt *hookOptions) handlePostTaskHookFailure(hookErr error) error {
	if opt.hookType == apis.PostBackupHook {
		backupOutput := &restic.BackupOutput{
			HostBackupStats: []v1beta1.HostBackupStats{
				{
					Hostname: opt.hostname,
					Phase:    v1beta1.HostBackupFailed,
					Error:    hookErr.Error(),
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
			HostRestoreStats: []v1beta1.HostRestoreStats{
				{
					Hostname: opt.hostname,
					Phase:    v1beta1.HostRestoreFailed,
					Error:    hookErr.Error(),
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

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

	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	prober "kmodules.xyz/prober/api/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"

)

type hookOptions struct {
	masterURL               string
	kubeConfigPath          string
	namespace               string
	hookType                string
	backupConfigurationName string
	restoreSessionName      string
	config                  *rest.Config
	kubeClient              kubernetes.Interface
	stashClient             cs.Interface
	appClient  appcatalog_cs.Interface
}

func NewCmdRunHook() *cobra.Command {
	opt := hookOptions{
		masterURL:      "",
		kubeConfigPath: "",
		namespace:      meta.Namespace(),
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

			err := opt.executeHook()
			if err != nil {
				if opt.hookType == apis.PreBackupHook || opt.hookType == apis.PreRestoreHook {

				}
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&opt.masterURL, "master", opt.masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.kubeConfigPath, "kubeconfig", opt.kubeConfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.backupConfigurationName, "backupconfiguration", opt.backupConfigurationName, "Name of the respective BackupConfiguration object")
	cmd.Flags().StringVar(&opt.restoreSessionName, "restoresession", opt.restoreSessionName, "Name of the respective RetsoreSession")
	cmd.Flags().StringVar(&opt.hookType, "hook-type", opt.restoreSessionName, "Type of hook to execute")
	return cmd
}

func (opt *hookOptions) executeHook() error {
	var backupConfig *v1beta1.BackupConfiguration
	var restoreSession *v1beta1.RestoreSession
	var hook *prober.Handler
	var err error

	if opt.backupConfigurationName != "" {
		backupConfig, err = opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(opt.backupConfigurationName, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	if opt.restoreSessionName != "" {
		restoreSession, err = opt.stashClient.StashV1beta1().RestoreSessions(opt.namespace).Get(opt.restoreSessionName, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	hook, err = opt.getHook(backupConfig, restoreSession)
	if err != nil {
		return err
	}

	podName, err := opt.getPodName(backupConfig, restoreSession)
	if err != nil {
		return err
	}
	return util.ExecuteHook(opt.config, hook, opt.hookType, podName, opt.namespace)
}

func (opt *hookOptions) getHook(backupConfig *v1beta1.BackupConfiguration, restoreSession *v1beta1.RestoreSession) (*prober.Handler, error) {
	switch opt.hookType {
	case apis.PreBackupHook:
		if backupConfig != nil && backupConfig.Spec.Hooks != nil && backupConfig.Spec.Hooks.PreBackup != nil {
			return backupConfig.Spec.Hooks.PreBackup, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in BackupConfiguration %s/%s", opt.hookType, opt.namespace, opt.backupConfigurationName)
		}
	case apis.PostBackupHook:
		if backupConfig != nil && backupConfig.Spec.Hooks != nil && backupConfig.Spec.Hooks.PostBackup != nil {
			return backupConfig.Spec.Hooks.PostBackup, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in BackupConfiguration %s/%s", opt.hookType, opt.namespace, opt.backupConfigurationName)
		}
	case apis.PreRestoreHook:
		if restoreSession != nil && restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PreRestore != nil {
			return restoreSession.Spec.Hooks.PreRestore, nil
		} else {
			return nil, fmt.Errorf("no %s hook found in RestoreSession %s/%s", opt.hookType, opt.namespace, opt.restoreSessionName)
		}
	case apis.PostRestoreHook:
		if restoreSession != nil && restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PostRestore != nil {
			return restoreSession.Spec.Hooks.PostRestore, nil
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
	if backupConfig!=nil&& backupConfig.Spec.Target !=nil{
		targetRef = backupConfig.Spec.Target.Ref
	} else if restoreSession!=nil && restoreSession.Spec.Target !=nil{
		targetRef = restoreSession.Spec.Target.Ref
	}else{
		return "",fmt.Errorf("invalid target. target can't be nil for executing hook in Function-Task model")
	}

	// if target is AppBinding, the desired pod will be the one of the pod selected by the endpoint of respective service.
	switch targetRef.Kind {
	case apis.KindAppBinding:
		return opt.getAppPodName(targetRef.Name)
	default:
		return os.Getenv(util.KeyPodName),nil
	}
}

func (opt *hookOptions)getAppPodName(appbindingName string)(string,error)  {
// get the AppBinding
appbinding,err :=opt.appClient.AppcatalogV1alpha1().AppBindings(opt.namespace).Get(appbindingName,metav1.GetOptions{})
if err!=nil{
	return "",err
}
if appbinding.Spec.ClientConfig.Service!=nil{
	endPoints:=appbinding.Spec.ClientConfig.Service.Name
}
}
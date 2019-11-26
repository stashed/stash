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
	"strings"
	"time"

	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
)

type options struct {
	invokerName      string
	namespace        string
	k8sClient        kubernetes.Interface
	stashClient      cs.Interface
	appcatalogClient appcatalog_cs.Interface
	ocClient         oc_cs.Interface
}

func NewCmdCreateBackupSession() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string

		opt = options{
			namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "create-backupsession",
		Short:             "create a BackupSession",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.k8sClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.appcatalogClient = appcatalog_cs.NewForConfigOrDie(config)
			// if cluster has OpenShift DeploymentConfig then generate OcClient
			if discovery.IsPreferredAPIResource(opt.k8sClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
				opt.ocClient = oc_cs.NewForConfigOrDie(config)
			}

			if opt.Invoker() != nil {
				log.Fatal(err)
			}
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.invokerName, "invokername", "", "Name of the respective BackupConfiguration/BackupBatch object")
	cmd.Flags().StringVar(&opt.namespace, "invokernamespace", opt.namespace, "Namespace of the respective BackupConfiguration object")

	return cmd
}

func (opt *options) createBackupSession(labels map[string]string, ref *core.ObjectReference) error {
	bsMeta := metav1.ObjectMeta{
		// Name format: <BackupConfiguration/BackupBatch name>-<timestamp in unix format>
		Name:            meta.ValidNameWithSuffix(ref.Name, fmt.Sprintf("%d", time.Now().Unix())),
		Namespace:       opt.namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}

	// create BackupSession
	_, _, err := v1beta1_util.CreateOrPatchBackupSession(opt.stashClient.StashV1beta1(), bsMeta, func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		// Set BackupConfiguration/BackupBatch as BackupSession Owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Spec.Invoker = api_v1beta1.BackupInvokerRef{
			APIGroup: api_v1beta1.SchemeGroupVersion.Group,
			Kind:     ref.Kind,
			Name:     ref.Name,
		}

		in.Labels = labels
		// add BackupConfiguration/BackupBatch name and kind as a labels so that BackupSession controller inside sidecar can discover this BackupSession
		in.Labels[util.LabelInvokerName] = ref.Name
		in.Labels[util.LabelInvokerType] = strings.ToLower(ref.Kind)

		return in
	})
	return err
}

func (opt *options) Invoker() error {
	wc := util.WorkloadClients{
		KubeClient:       opt.k8sClient,
		StashClient:      opt.stashClient,
		AppCatalogClient: opt.appcatalogClient,
		OcClient:         opt.ocClient,
	}

	backupConfiguration, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(opt.invokerName, metav1.GetOptions{})
	if err == nil && !kerr.IsNotFound(err) {
		// create BackupConfiguration reference to set BackupSession owner
		ref, err := reference.GetReference(stash_scheme.Scheme, backupConfiguration)
		if err != nil {
			return err
		}
		// if target does not exist then skip creating BackupSession
		if backupConfiguration.Spec.Target != nil && !wc.IsTargetExist(backupConfiguration.Spec.Target.Ref, backupConfiguration.Namespace) {
			msg := fmt.Sprintf("Skipping creating BackupSession. Reason: Target workload %s/%s does not exist.",
				strings.ToLower(backupConfiguration.Spec.Target.Ref.Kind), backupConfiguration.Spec.Target.Ref.Name)
			log.Infoln(msg)

			// write event to BackupConfiguration denoting that backup session has been skipped
			return writeBackupSessionSkippedEvent(opt.k8sClient, ref, msg)
		}
		err = opt.createBackupSession(backupConfiguration.OffshootLabels(), ref)
		if err != nil {
			return err
		}
	}

	if err == nil && kerr.IsNotFound(err) {
		backupBatch, err := opt.stashClient.StashV1beta1().BackupBatches(opt.namespace).Get(opt.invokerName, metav1.GetOptions{})
		if err == nil && !kerr.IsNotFound(err) {
			// create backupBatch reference to set BackupSession owner
			ref, err := reference.GetReference(stash_scheme.Scheme, backupBatch)
			if err != nil {
				return err
			}
			// if target does not exist then skip creating BackupSession
			for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
				if backupConfigTemp.Spec.Target != nil && !wc.IsTargetExist(backupConfigTemp.Spec.Target.Ref, backupConfigTemp.Namespace) {
					msg := fmt.Sprintf("Skipping creating BackupSession. Reason: Target workload %s/%s does not exist.",
						strings.ToLower(backupConfigTemp.Spec.Target.Ref.Kind), backupConfigTemp.Spec.Target.Ref.Name)
					log.Infoln(msg)

					// write event to BackupConfiguration denoting that backup session has been skipped
					return writeBackupSessionSkippedEvent(opt.k8sClient, ref, msg)
				}
			}
			err = opt.createBackupSession(backupBatch.OffshootLabels(), ref)
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}
	return err
}

func writeBackupSessionSkippedEvent(kubeClient kubernetes.Interface, ref *core.ObjectReference, msg string) error {
	_, err := eventer.CreateEvent(
		kubeClient,
		eventer.EventSourceBackupTriggeringCronJob,
		ref,
		core.EventTypeNormal,
		eventer.EventReasonBackupSkipped,
		msg,
	)
	return err
}

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

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
)

type options struct {
	invokerKind      string
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
			namespace: meta.PodNamespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "create-backupsession",
		Short:             "create a BackupSession",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				klog.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.k8sClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.appcatalogClient = appcatalog_cs.NewForConfigOrDie(config)
			// if cluster has OpenShift DeploymentConfig then generate OcClient
			if discovery.IsPreferredAPIResource(opt.k8sClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
				opt.ocClient = oc_cs.NewForConfigOrDie(config)
			}

			if err = opt.createBackupSession(); err != nil {
				klog.Fatal(err)
			}
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.invokerName, "invoker-name", "", "Name of the invoker")
	cmd.Flags().StringVar(&opt.invokerKind, "invoker-kind", opt.invokerKind, "Type of the backup invoker")

	return cmd
}

func (opt *options) createBackupSession() error {
	inv, err := invoker.NewBackupInvoker(opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
	if err != nil {
		return err
	}

	retryLeft := int32(0)
	retryConfig := inv.GetRetryConfig()
	if retryConfig != nil {
		retryLeft = retryConfig.MaxRetry
	}

	session := inv.NewSession()
	_, _, err = v1beta1_util.CreateOrPatchBackupSession(
		context.TODO(),
		opt.stashClient.StashV1beta1(),
		session.ObjectMeta,
		func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
			core_util.EnsureOwnerReference(&in.ObjectMeta, inv.GetOwnerRef())
			in.Labels = session.Labels
			in.Spec = session.Spec
			in.Spec.RetryLeft = retryLeft
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

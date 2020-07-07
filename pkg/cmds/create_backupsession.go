/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

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
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
)

type options struct {
	invokerType      string
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

			if err = opt.createBackupSession(); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.invokerName, "invoker-name", "", "Name of the invoker")
	cmd.Flags().StringVar(&opt.invokerType, "invoker-type", opt.invokerType, "Type of the backup invoker")

	return cmd
}

func (opt *options) createBackupSession() error {
	invoker, err := apis.ExtractBackupInvokerInfo(opt.stashClient, opt.invokerType, opt.invokerName, opt.namespace)
	if err != nil {
		return err
	}

	bsMeta := metav1.ObjectMeta{
		// Name format: <invoker name>-<timestamp in unix format>
		Name:            meta.NameWithSuffix(opt.invokerName, fmt.Sprintf("%d", time.Now().Unix())),
		Namespace:       opt.namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}

	// create BackupSession
	_, _, err = v1beta1_util.CreateOrPatchBackupSession(
		context.TODO(),
		opt.stashClient.StashV1beta1(),
		bsMeta,
		func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
			// Set BackupConfiguration  as BackupSession Owner
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
			in.Spec.Invoker = api_v1beta1.BackupInvokerRef{
				APIGroup: api_v1beta1.SchemeGroupVersion.Group,
				Kind:     opt.invokerType,
				Name:     opt.invokerName,
			}

			in.Labels = invoker.Labels
			// Add invoker name and kind as a labels so that BackupSession controller inside sidecar can discover this BackupSession
			in.Labels[apis.LabelInvokerName] = opt.invokerName
			in.Labels[apis.LabelInvokerType] = opt.invokerType

			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

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
	"time"

	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	storage_api_v1 "k8s.io/api/storage/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	prober "kmodules.xyz/prober/probe"
)

func NewCmdRestoreVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
			metrics: restic.MetricsOptions{
				Enabled: true,
				JobName: "stash-volumesnapshot-restorer",
			},
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-vs",
		Short:             "Restore PVC from VolumeSnapshot",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.config = config
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = vs_cs.NewForConfigOrDie(config)

			restoreOutput, err := opt.restoreVolumeSnapshot()
			if err != nil {
				return err
			}
			statOpt := status.UpdateStatusOptions{
				Config:         config,
				KubeClient:     opt.kubeClient,
				StashClient:    opt.stashClient,
				Namespace:      opt.namespace,
				RestoreSession: opt.restoresession,
				Metrics:        opt.metrics,
			}
			return statOpt.UpdatePostRestoreStatus(restoreOutput)
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.restoresession, "restoresession", "", "Name of the respective RestoreSession object")
	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	return cmd
}

func (opt *VSoption) restoreVolumeSnapshot() (*restic.RestoreOutput, error) {
	// start clock to measure the time takes to restore the volumes
	startTime := time.Now()

	restoreSession, err := opt.stashClient.StashV1beta1().RestoreSessions(opt.namespace).Get(opt.restoresession, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if restoreSession.Spec.Target == nil {
		return nil, fmt.Errorf("no target has been specified for RestoreSession %s/%s", restoreSession.Namespace, restoreSession.Name)
	}

	// If preRestore hook is specified, then execute those hooks first
	if restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PreRestore != nil {
		log.Infoln("Executing preRestore hooks........")
		podName := os.Getenv(apis.KeyPodName)
		if podName == "" {
			return nil, fmt.Errorf("failed to execute preRestore hooks. Reason: POD_NAME environment variable not found")
		}
		err := prober.RunProbe(opt.config, restoreSession.Spec.Hooks.PreRestore, podName, opt.namespace)
		if err != nil {
			return nil, err
		}
		log.Infoln("preRestore hooks has been executed successfully")
	}

	var pvcList []core.PersistentVolumeClaim
	// if replica field is specified, then use it. otherwise, default it to 1
	replicas := int32(1)
	if restoreSession.Spec.Target.Replicas != nil {
		replicas = *restoreSession.Spec.Target.Replicas
	}

	// resolve the volumeClaimTemplates and prepare PVC definiton
	for ordinal := int32(0); ordinal < replicas; ordinal++ {
		pvcs, err := resolve.GetPVCFromVolumeClaimTemplates(ordinal, restoreSession.Spec.Target.VolumeClaimTemplates)
		if err != nil {
			return nil, err
		}
		pvcList = append(pvcList, pvcs...)
	}

	// createdPVCs holds the definition of the PVCs that has been created successfully
	var createdPVCs []core.PersistentVolumeClaim

	// now create the PVCs
	restoreOutput := &restic.RestoreOutput{}
	for i := range pvcList {
		// verify that the respective VolumeSnapshot exist
		if pvcList[i].Spec.DataSource != nil {
			_, err = opt.snapshotClient.SnapshotV1beta1().VolumeSnapshots(opt.namespace).Get(pvcList[i].Spec.DataSource.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) { // respective VolumeSnapshot does not exist
					restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
						Hostname: pvcList[i].Name,
						Phase:    api_v1beta1.HostRestoreFailed,
						Error:    fmt.Sprintf("VolumeSnapshot %s/%s does not exist", pvcList[i].Namespace, pvcList[i].Spec.DataSource.Name),
					})
					// continue to process next VolumeSnapshot
					continue
				} else {
					return nil, err
				}
			}
		}

		// now, create the PVC
		pvc, err := opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Create(&pvcList[i])
		if err != nil {
			if kerr.IsAlreadyExists(err) {
				restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
					Hostname: pvcList[i].Name,
					Phase:    api_v1beta1.HostRestoreFailed,
					Error:    fmt.Sprintf("Failed to create PVC %s/%s. Reason: PVC already exist.", pvcList[i].Namespace, pvcList[i].Name),
				})
				// continue to process next pvc
				continue
			} else {
				return nil, err
			}
		}
		// PVC has been created successfully. store it's definition so that we can wait for it to be initialized
		createdPVCs = append(createdPVCs, *pvc)
	}

	// now, wait for the PVCs to be initialized from respective VolumeSnapshot
	for i := range createdPVCs {
		// find out the storage class that has been used in this PVC. We need to know it's binding mode to decide whether we should wait
		// for it to be bound with the respective PV.
		storageClass, err := opt.kubeClient.StorageV1().StorageClasses().Get(types.String(createdPVCs[i].Spec.StorageClassName), metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) { // storage class not found. so, restore won't be completed.
				restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
					Hostname: createdPVCs[i].Name,
					Phase:    api_v1beta1.HostRestoreFailed,
					Error:    fmt.Sprintf("failed to restore. Reason: StorageClass %s not found.", *createdPVCs[i].Spec.StorageClassName),
				})
				// continue to process next pvc
				continue
			} else {
				return nil, err
			}
		}
		// don't wait for a PVC that uses "WaitForFirstConsumer" binding mode.
		// this PVC will be bounded when a workload will use it.
		if *storageClass.VolumeBindingMode == storage_api_v1.VolumeBindingWaitForFirstConsumer {
			restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
				Hostname: createdPVCs[i].Name,
				Phase:    api_v1beta1.HostRestoreUnknown,
				Error: fmt.Sprintf("Stash is unable to verify whether the volume has been initialized from snapshot data or not." +
					"Reason: volume binding mode is WaitForFirstConsumer."),
			})
			// continue to process next pvc
			continue
		}

		// wait for the PVC to be bound with the respective PV
		err = util.WaitUntilPVCReady(opt.kubeClient, createdPVCs[i].ObjectMeta)
		if err != nil {
			restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
				Hostname: createdPVCs[i].Name,
				Phase:    api_v1beta1.HostRestoreFailed,
				Error:    fmt.Sprintf("failed to restore the volume. Reason: %v", err),
			})
			// continue to process next pvc
			continue
		}
		// restore completed for this PVC.
		restoreOutput.HostRestoreStats = append(restoreOutput.HostRestoreStats, api_v1beta1.HostRestoreStats{
			Hostname: createdPVCs[i].Name,
			Phase:    api_v1beta1.HostRestoreSucceeded,
			Duration: time.Since(startTime).String(),
		})
	}
	// If postRestore hook is specified, then execute those hooks after restore
	if restoreSession.Spec.Hooks != nil && restoreSession.Spec.Hooks.PostRestore != nil {
		log.Infoln("Executing postRestore hooks........")
		podName := os.Getenv(apis.KeyPodName)
		if podName == "" {
			return nil, fmt.Errorf("failed to execute postRestore hook. Reason: POD_NAME environment variable not found")
		}
		err := prober.RunProbe(opt.config, restoreSession.Spec.Hooks.PostRestore, podName, opt.namespace)
		if err != nil {
			return nil, fmt.Errorf(err.Error() + "Warning: The actual restore process may be succeeded." +
				"Hence, the restored data might be present in the target even if the overall RestoreSession phase is 'Failed'")
		}
		log.Infoln("postRestore hooks has been executed successfully")
	}

	return restoreOutput, nil
}

package cmds

import (
	"fmt"
	"strconv"
	"time"

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
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdRestoreVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
			metrics: restic.MetricsOptions{
				Enabled: false,
			},
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-vs",
		Short:             "Restore PVC from VolumeSnapshot",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = vs_cs.NewForConfigOrDie(config)

			err = opt.RestoreVolumeSnapshot()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.name, "restoresession.name", "", "Set Restore Session Name")
	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	return cmd
}

func (opt *VSoption) RestoreVolumeSnapshot() error {
	// Start clock to measure total session duration
	startTime := time.Now()

	restoreSession, err := opt.stashClient.StashV1beta1().RestoreSessions(opt.namespace).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if restoreSession == nil {
		return fmt.Errorf("restoreSession is nil")
	}
	if restoreSession.Spec.Target == nil {
		return fmt.Errorf("restoreSession Target is nil")
	}

	pvcList := make([]core.PersistentVolumeClaim, 0)
	replicas := int32(1)

	if restoreSession.Spec.Target.Replicas != nil {
		replicas = *restoreSession.Spec.Target.Replicas
	}
	for ordinal := int32(0); ordinal < replicas; ordinal++ {
		pvcs, err := opt.getPVCFromVolumeClaimTemplates(ordinal, restoreSession.Spec.Target.VolumeClaimTemplates, startTime)
		if err != nil {
			return err
		}
		pvcList = append(pvcList, pvcs...)
	}

	pvcAllReadyExists := false
	volumeSnapshotExists := false

	for _, pvc := range pvcList {

		_, err = opt.snapshotClient.VolumesnapshotV1alpha1().VolumeSnapshots(opt.namespace).Get(pvc.Spec.DataSource.Name, metav1.GetOptions{})
		if err != nil {
			volumeSnapshotExists = true
			// write failure event for not existing volumeSnapshot
			restoreOutput := restic.RestoreOutput{
				HostRestoreStats: v1beta1.HostRestoreStats{
					Hostname: pvc.Name,
					Phase:    v1beta1.HostRestoreFailed,
					Error:    fmt.Sprintf("%s not exixts", pvc.Spec.DataSource.Name),
				},
			}
			err := opt.updateRestoreSessionStatus(restoreOutput, startTime)
			if err != nil {
				return err
			}
			continue
		}
		_, err = opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Get(pvc.Name, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				_, err := opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Create(&pvc)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// write failure event for existing PVC
			pvcAllReadyExists = true
			restoreOutput := restic.RestoreOutput{
				HostRestoreStats: v1beta1.HostRestoreStats{
					Hostname: pvc.Name,
					Phase:    v1beta1.HostRestoreFailed,
					Error:    fmt.Sprintf("%s already exixts", pvc.Name),
				},
			}
			err := opt.updateRestoreSessionStatus(restoreOutput, startTime)
			if err != nil {
				return err
			}
		}
	}

	if pvcAllReadyExists || volumeSnapshotExists {
		return nil
	}

	for _, pvc := range pvcList {
		storageClass, err := opt.kubeClient.StorageV1().StorageClasses().Get(types.String(pvc.Spec.StorageClassName), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if *storageClass.VolumeBindingMode != storage_api_v1.VolumeBindingImmediate {
			// write failure event because of VolumeBindingMode is WaitForFirstConsumer
			restoreOutput := restic.RestoreOutput{
				HostRestoreStats: v1beta1.HostRestoreStats{
					Hostname: pvc.Name,
					Phase:    v1beta1.HostRestoreUnknown,
					Error:    fmt.Sprintf("VolumeBindingMode is 'WaitForFirstConsumer'. Stash is unable to decide wheather the restore has succeeded or not as the PVC will not bind with respective PV until any workload mount it."),
				},
			}
			err := opt.updateRestoreSessionStatus(restoreOutput, startTime)
			if err != nil {
				return err
			}
			continue
		}

		err = util.WaitUntilPVCReady(opt.kubeClient, pvc.ObjectMeta)
		if err != nil {
			return err
		}
		restoreOutput := restic.RestoreOutput{
			HostRestoreStats: v1beta1.HostRestoreStats{
				Hostname: pvc.Name,
				Phase:    v1beta1.HostRestoreSucceeded,
			},
		}
		err = opt.updateRestoreSessionStatus(restoreOutput, startTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func (opt *VSoption) getPVCFromVolumeClaimTemplates(ordinal int32, claimTemplates []core.PersistentVolumeClaim, startTime time.Time) ([]core.PersistentVolumeClaim, error) {
	pvcList := make([]core.PersistentVolumeClaim, 0)

	for _, claim := range claimTemplates {
		pvc, err := opt.getPVCDefinition(ordinal, claim)
		if err != nil {
			// write failure event
			restoreOutput := restic.RestoreOutput{
				HostRestoreStats: v1beta1.HostRestoreStats{
					Hostname: pvc.Name,
					Phase:    v1beta1.HostRestoreFailed,
					Error:    err.Error(),
				},
			}
			err := opt.updateRestoreSessionStatus(restoreOutput, startTime)
			return pvcList, err
		}
		pvc.Namespace = opt.namespace
		pvcList = append(pvcList, pvc)
	}
	return pvcList, nil
}

func (opt *VSoption) getPVCDefinition(ordinal int32, claim core.PersistentVolumeClaim) (core.PersistentVolumeClaim, error) {
	inputs := make(map[string]string)
	inputs["POD_ORDINAL"] = strconv.Itoa(int(ordinal))
	dataSource := claim.Spec.DataSource
	err := resolve.ResolvePVCSpec(&claim, inputs)
	claim.Spec.DataSource = dataSource
	return claim, err
}

func (opt *VSoption) updateRestoreSessionStatus(restoreOutput restic.RestoreOutput, startTime time.Time) error {
	// Update Backup Session
	o := status.UpdateStatusOptions{
		KubeClient:     opt.kubeClient,
		StashClient:    opt.stashClient.(*cs.Clientset),
		Namespace:      opt.namespace,
		RestoreSession: opt.name,
	}
	// Volume Snapshot complete. Read current time and calculate total backup duration.
	endTime := time.Now()
	restoreOutput.HostRestoreStats.Duration = endTime.Sub(startTime).String()
	return o.UpdatePostRestoreStatus(&restoreOutput)
}

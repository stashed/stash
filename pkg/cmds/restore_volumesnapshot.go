package cmds

import (
	"fmt"
	"strconv"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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

type PVC struct {
	podOrdinal *int32
	pvcName    string
	pvc        corev1.PersistentVolumeClaim
}

func NewCmdRestoreVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
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

	pvcData := []PVC{}

	if restoreSession.Spec.Target.Replicas == nil {
		for _, vol := range restoreSession.Spec.Target.VolumeClaimTemplates {
			pvcData = append(pvcData, PVC{pvcName: vol.Name, pvc: vol})

		}
	} else {
		for i := int32(0); i < *restoreSession.Spec.Target.Replicas; i++ {
			for _, vol := range restoreSession.Spec.Target.VolumeClaimTemplates {
				pvcData = append(pvcData, PVC{pvcName: vol.Name, podOrdinal: types.Int32P(i), pvc: vol})
			}
		}

	}

	objectMeta := []metav1.ObjectMeta{}
	pvcAllReadyExists := false

	for _, data := range pvcData {
		pvc := opt.getPVCDefinition(data)

		_, err = opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Get(pvc.Name, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				pvc, err := opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Create(pvc)
				if err != nil {
					return err
				}
				objectMeta = append(objectMeta, pvc.ObjectMeta)
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
	if pvcAllReadyExists {
		return nil
	}

	for i, data := range pvcData {
		storageClass, err := opt.kubeClient.StorageV1().StorageClasses().Get(types.String(data.pvc.Spec.StorageClassName), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if *storageClass.VolumeBindingMode != storage_api_v1.VolumeBindingImmediate {
			// write failure event because of VolumeBindingMode is WaitForFirstConsumer
			restoreOutput := restic.RestoreOutput{
				HostRestoreStats: v1beta1.HostRestoreStats{
					Hostname: objectMeta[i].Name,
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

		err = util.WaitUntilPVCReady(opt.kubeClient, objectMeta[i])
		if err != nil {
			return err
		}
		restoreOutput := restic.RestoreOutput{
			HostRestoreStats: v1beta1.HostRestoreStats{
				Hostname: objectMeta[i].Name,
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

func (opt *VSoption) getPVCDefinition(data PVC) *corev1.PersistentVolumeClaim {
	inputs := make(map[string]string, 0)
	if data.podOrdinal == nil {
		data.pvc.Name = data.pvcName
	} else {
		data.pvc.Name = fmt.Sprintf("%v-%v", data.pvcName, *data.podOrdinal)
		inputs["POD_ORDINAL"] = strconv.Itoa(int(*data.podOrdinal))
	}
	inputs["CLAIM_NAME"] = data.pvcName
	err := resolve.ResolvePVCSpec(&data.pvc, inputs)
	if err != nil {
		return nil
	}
	return &data.pvc
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

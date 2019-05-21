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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/apis/core"
	"kmodules.xyz/client-go/meta"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
)

type PVC struct {
	podOrdinal *int32
	pvcName    string
	pvc        *corev1.PersistentVolumeClaim
}

var (
	pvcData = []PVC{}
)

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
		Short:             "Take a restore snapshot of PersistentVolumeClaims",
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

	if restoreSession == nil || restoreSession.Spec.Target == nil {
		return fmt.Errorf("restoreSession or  restoreSession target is nil")
	}
	fmt.Println("target", restoreSession.Spec.Target)

	if restoreSession.Spec.Target.Replicas == nil {
		pvcData = []PVC{}
		for _, vol := range restoreSession.Spec.Target.VolumeClaimTemplates {
			pvcData = append(pvcData, PVC{podOrdinal: types.Int32P(0), pvcName: vol.Name, pvc: &vol})

		}
	} else {
		pvcData = []PVC{}
		for i := int32(0); i < types.Int32(restoreSession.Spec.Target.Replicas); i++ {
			for _, vol := range restoreSession.Spec.Target.VolumeClaimTemplates {
				pvcData = append(pvcData, PVC{pvcName: vol.Name, podOrdinal: types.Int32P(i), pvc: &vol})
			}
		}

	}

	objectMeta := []metav1.ObjectMeta{}

	for _, data := range pvcData {
		pvc := opt.getPVCDefinition(data)
		_, err := opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Get(data.pvc.Name, metav1.GetOptions{})
		if err != nil {
			pvc, err := opt.kubeClient.CoreV1().PersistentVolumeClaims(opt.namespace).Create(pvc)
			if err != nil {
				return err
			}
			objectMeta = append(objectMeta, pvc.ObjectMeta)

		} else {
			// write failure event
			return opt.setRestoreSessionFailed(restoreSession, fmt.Sprintf("PVC %s is already exists", data.pvc.Name))
		}
	}

	for i, pvc := range pvcData {
		storageClass, err := opt.kubeClient.StorageV1().StorageClasses().Get(types.String(pvc.pvc.Spec.StorageClassName), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if *storageClass.VolumeBindingMode != storage_api_v1.VolumeBindingImmediate {
			// write failure event
			return opt.setRestoreSessionFailed(restoreSession, fmt.Sprintf("VolumeBindingMode is equal to %s because of phase is Pending", storage_api_v1.VolumeBindingWaitForFirstConsumer))
		}
		err = util.WaitUntilPVCReady(opt.kubeClient, objectMeta[i])
		if err != nil {
			return err
		}
		// Update Backup Session
		o := status.UpdateStatusOptions{
			KubeClient:     opt.kubeClient,
			StashClient:    opt.stashClient.(*cs.Clientset),
			Namespace:      opt.namespace,
			RestoreSession: opt.name,
		}
		restoreOutput := restic.RestoreOutput{
			HostRestoreStats: v1beta1.HostRestoreStats{
				Hostname: pvc.pvcName,
				Phase:    v1beta1.HostRestoreSucceeded,
			},
		}
		// Volume Snapshot complete. Read current time and calculate total backup duration.
		endTime := time.Now()
		restoreOutput.HostRestoreStats.Duration = endTime.Sub(startTime).String()
		err = o.UpdatePostRestoreStatus(&restoreOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func (opt *VSoption) getPVCDefinition(data PVC) *corev1.PersistentVolumeClaim {
	data.pvc.Name = fmt.Sprintf("%v-%v", data.pvcName, types.Int32(data.podOrdinal))
	inputs := make(map[string]string, 0)
	inputs["PVC_NAME"] = data.pvcName
	fmt.Println(strconv.Itoa(int(types.Int32(data.podOrdinal))))
	inputs["POD_ORDINAL"] = strconv.Itoa(int(types.Int32(data.podOrdinal)))
	err := resolve.ResolvePVCSpec(data.pvc, inputs)
	if err != nil {
		return nil
	}
	return data.pvc
}

func (opt *VSoption) setRestoreSessionFailed(restoreSession *v1beta1.RestoreSession, message string) error {
	// set RestoreSession phase to "Failed"
	_, err := v1beta1_util.UpdateRestoreSessionStatus(opt.stashClient.StashV1beta1(), restoreSession, func(in *v1beta1.RestoreSessionStatus) *v1beta1.RestoreSessionStatus {
		in.Phase = v1beta1.RestoreSessionFailed
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}
	// write failure event
	_, err = eventer.CreateEvent(
		opt.kubeClient,
		eventer.RestoreSessionEventComponent,
		restoreSession,
		core.EventTypeWarning,
		eventer.EventReasonRestoreSessionFailed,
		message,
	)
	return err
}

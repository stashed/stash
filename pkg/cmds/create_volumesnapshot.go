package cmds

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapshot_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	workload_api "kmodules.xyz/webhook-runtime/apis/workload/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
)

var (
	pvclist = make([]string, 0)
)

type VSoption struct {
	name           string
	namespace      string
	kubeClient     kubernetes.Interface
	stashClient    cs.Interface
	snapshotClient snapshot_cs.Interface
}

func NewCmdCreateVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "create-vs",
		Short:             "Take a backup of Volume Snapshot",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = snapshot_cs.NewForConfigOrDie(config)

			err = opt.CreateBackupSessionVolumeSnapshot()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.name, "backupsession.name", "", "Set BackupSession Name")
	cmd.Flags().StringVar(&opt.namespace, "backupsession.namespace", opt.namespace, "Set BackupSession Namespace")
	return cmd
}

func (opt *VSoption) CreateBackupSessionVolumeSnapshot() error {
	backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	backupConfiguration, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(backupSession.Spec.BackupConfiguration.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backupConfiguration == nil {
		return fmt.Errorf("BackupConfiguration is nil")
	}
	kind := backupConfiguration.Spec.Target.Ref.Kind
	name := backupConfiguration.Spec.Target.Ref.Name
	namespace := backupConfiguration.Namespace

	switch kind {
	case workload_api.KindDeployment:
		deployment, err := opt.kubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		vollist := deployment.Spec.Template.Spec.Volumes
		totalPVC(vollist)
	case workload_api.KindDaemonSet:
		daemon, err := opt.kubeClient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		vollist := daemon.Spec.Template.Spec.Volumes
		totalPVC(vollist)
	case workload_api.KindReplicationController:
		RC, err := opt.kubeClient.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		vollist := RC.Spec.Template.Spec.Volumes
		totalPVC(vollist)
	case workload_api.KindReplicaSet:
		RS, err := opt.kubeClient.AppsV1().ReplicaSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		vollist := RS.Spec.Template.Spec.Volumes
		totalPVC(vollist)
	case workload_api.KindStatefulSet:
		SS, err := opt.kubeClient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		pvclist = []string{}
		vollist := SS.Spec.VolumeClaimTemplates
		for i := int32(0); i < types.Int32(SS.Spec.Replicas); i++ {
			podName := fmt.Sprintf("%v-%v", SS.Name, i)
			for _, list := range vollist {
				pvclist = append(pvclist, fmt.Sprintf("%v-%v", list.Name, podName))
			}
		}

	case apis.KindPersistentVolumeClaim:
		pvclist = []string{}
		pvcName := backupConfiguration.Spec.Target.Ref.Name
		pvclist = append(pvclist, pvcName)
	}

	ObjectMeta := []metav1.ObjectMeta{}

	for _, pvcName := range pvclist {
		volumesnapshot := opt.VolumeSnapshot(backupConfiguration, pvcName)
		vs, err := opt.snapshotClient.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Create(&volumesnapshot)
		if err != nil {
			return err
		}
		ObjectMeta = append(ObjectMeta, vs.ObjectMeta)
	}

	for i, pvcName := range pvclist {
		// Start clock to measure total session duration
		startTime := time.Now()
		err = util.WaitUntilVolumeSnapshotReady(opt.snapshotClient, ObjectMeta[i])
		if err != nil {
			return err
		}
		// Update Backup Session
		o := status.UpdateStatusOptions{
			KubeClient:    opt.kubeClient,
			StashClient:   opt.stashClient.(*cs.Clientset),
			Namespace:     opt.namespace,
			BackupSession: opt.name,
		}
		backupOutput := restic.BackupOutput{
			HostBackupStats: v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    v1beta1.HostBackupSucceeded,
			},
		}
		// Volume Snapshot complete. Read current time and calculate total backup duration.
		endTime := time.Now()
		backupOutput.HostBackupStats.Duration = endTime.Sub(startTime).String()

		err = o.UpdatePostBackupStatus(&backupOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func (opt *VSoption) VolumeSnapshot(backupConfiguration *v1beta1.BackupConfiguration, pvcName string) (volumeSnapshot crdv1.VolumeSnapshot) {
	curTime := fmt.Sprintf("%d-%02d-%02dt%02d-%02d-%02d-00-00-stash",
		time.Now().Year(), time.Now().Month(), time.Now().Day(),
		time.Now().Hour(), time.Now().Minute(), time.Now().Second())

	return crdv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(pvcName + "-" + curTime),
			Namespace: backupConfiguration.Namespace,
		},
		Spec: crdv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &backupConfiguration.Spec.Target.VolumeSnapshotClassName,
			Source: &corev1.TypedLocalObjectReference{
				Kind: apis.KindPersistentVolumeClaim,
				Name: pvcName,
			},
		},
	}

}

func totalPVC(vollist []corev1.Volume) {
	pvclist = []string{}
	for _, list := range vollist {
		if list.PersistentVolumeClaim != nil {
			pvclist = append(pvclist, list.PersistentVolumeClaim.ClaimName)
		}
	}
}

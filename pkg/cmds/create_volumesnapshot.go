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
	"fmt"
	"os"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/pkg/volumesnapshot"

	vs "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/meta"
	prober "kmodules.xyz/prober/probe"
)

type VSoption struct {
	backupsession string

	namespace      string
	config         *rest.Config
	kubeClient     kubernetes.Interface
	stashClient    cs.Interface
	snapshotClient vs_cs.Interface
	metrics        restic.MetricsOptions

	// Invoker
	invokerKind string
	invokerName string

	//Target
	targetName string
	targetKind string
}

func NewCmdCreateVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
			metrics: restic.MetricsOptions{
				Enabled: true,
			},
		}
	)

	cmd := &cobra.Command{
		Use:               "create-vs",
		Short:             "Take snapshot of PersistentVolumeClaims",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				klog.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.config = config
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = vs_cs.NewForConfigOrDie(config)

			// get backup session
			backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(context.TODO(), opt.backupsession, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// get backup Invoker
			inv, err := invoker.ExtractBackupInvokerInfo(opt.stashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, backupSession.Namespace)
			if err != nil {
				return err
			}

			opt.metrics.JobName = fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.TypeMeta.Kind), inv.ObjectMeta.Namespace, inv.ObjectMeta.Name)

			for _, targetInfo := range inv.TargetsInfo {
				if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {
					backupOutput, err := opt.createVolumeSnapshot(backupSession.ObjectMeta, inv, targetInfo)
					if err != nil {
						return err
					}

					statOpt := status.UpdateStatusOptions{
						Config:        config,
						KubeClient:    opt.kubeClient,
						StashClient:   opt.stashClient,
						Namespace:     opt.namespace,
						BackupSession: opt.backupsession,
						Metrics:       opt.metrics,
					}
					statOpt.TargetRef.Name = opt.targetName
					statOpt.TargetRef.Kind = opt.targetKind

					return statOpt.UpdatePostBackupStatus(backupOutput, inv, targetInfo)

				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.targetName, "target-name", opt.targetName, "Name of the Target")
	cmd.Flags().StringVar(&opt.targetKind, "target-kind", opt.targetKind, "Kind of the Target")
	cmd.Flags().StringVar(&opt.backupsession, "backupsession", "", "Name of the respective BackupSession object")
	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	return cmd
}

func (opt *VSoption) createVolumeSnapshot(bsMeta metav1.ObjectMeta, inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo) (*restic.BackupOutput, error) {
	// Start clock to measure total session duration
	startTime := time.Now()

	if targetInfo.Target == nil {
		return nil, fmt.Errorf("no target has been specified for Backup invoker %s", targetInfo.Target.Ref.Name)
	}

	// If preBackup hook is specified, then execute those hooks first
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreBackup != nil {
		klog.Infoln("Executing preBackup hooks........")
		podName := os.Getenv(apis.KeyPodName)
		if podName == "" {
			return nil, fmt.Errorf("failed to execute preBackup hooks. Reason: POD_NAME environment variable not found")
		}
		err := prober.RunProbe(opt.config, targetInfo.Hooks.PreBackup, podName, opt.namespace)
		if err != nil {
			return nil, err
		}
		klog.Infoln("preBackup hooks has been executed successfully")
	}

	pvcNames, err := opt.getTargetPVCNames(targetInfo.Target.Ref, targetInfo.Target.Replicas)
	if err != nil {
		return nil, err
	}

	vsMeta := []metav1.ObjectMeta{}

	// create VolumeSnapshots
	for _, pvcName := range pvcNames {
		// use timestamp suffix of BackupSession name as suffix of the VolumeSnapshots name
		parts := strings.Split(bsMeta.Name, "-")
		volumeSnapshot := opt.getVolumeSnapshotDefinition(targetInfo.Target, inv.ObjectMeta.Namespace, pvcName, parts[len(parts)-1])
		snapshot, err := opt.snapshotClient.SnapshotV1beta1().VolumeSnapshots(opt.namespace).Create(context.TODO(), &volumeSnapshot, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		vsMeta = append(vsMeta, snapshot.ObjectMeta)
	}

	// now wait for all the VolumeSnapshots are completed (ready to to use)
	backupOutput := &restic.BackupOutput{
		BackupTargetStatus: api_v1beta1.BackupTargetStatus{
			Ref: targetInfo.Target.Ref,
		},
	}
	for i, pvcName := range pvcNames {
		// wait until this VolumeSnapshot is ready to use
		err = util.WaitUntilVolumeSnapshotReady(opt.snapshotClient, vsMeta[i])
		if err != nil {
			backupOutput.BackupTargetStatus.Stats = append(backupOutput.BackupTargetStatus.Stats, api_v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    api_v1beta1.HostBackupFailed,
				Error:    err.Error(),
			})
		} else {
			backupOutput.BackupTargetStatus.Stats = append(backupOutput.BackupTargetStatus.Stats, api_v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    api_v1beta1.HostBackupSucceeded,
				Duration: time.Since(startTime).String(),
			})
		}

	}
	err = volumesnapshot.CleanupSnapshots(inv.RetentionPolicy, backupOutput.BackupTargetStatus.Stats, bsMeta.Namespace, opt.snapshotClient)
	if err != nil {
		return nil, err
	}

	// If postBackup hook is specified, then execute those hooks after backup
	if targetInfo.Hooks != nil && targetInfo.Hooks.PostBackup != nil {
		klog.Infoln("Executing postBackup hooks........")
		podName := os.Getenv(apis.KeyPodName)
		if podName == "" {
			return nil, fmt.Errorf("failed to execute postBackup hook. Reason: POD_NAME environment variable not found")
		}
		err := prober.RunProbe(opt.config, targetInfo.Hooks.PostBackup, podName, opt.namespace)
		if err != nil {
			return nil, fmt.Errorf(err.Error() + "Warning: The actual backup process may be succeeded." +
				"Hence, the backup snapshots might be present in the backend even if the overall BackupSession phase is 'Failed'")
		}
		klog.Infoln("postBackup hooks has been executed successfully")
	}

	return backupOutput, nil
}

func (opt *VSoption) getTargetPVCNames(targetRef api_v1beta1.TargetRef, replicas *int32) ([]string, error) {
	var pvcList []string

	switch targetRef.Kind {
	case apis.KindDeployment:
		deployment, err := opt.kubeClient.AppsV1().Deployments(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(deployment.Spec.Template.Spec.Volumes)

	case apis.KindDaemonSet:
		daemon, err := opt.kubeClient.AppsV1().DaemonSets(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(daemon.Spec.Template.Spec.Volumes)

	case apis.KindReplicationController:
		rc, err := opt.kubeClient.CoreV1().ReplicationControllers(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(rc.Spec.Template.Spec.Volumes)

	case apis.KindReplicaSet:
		rs, err := opt.kubeClient.AppsV1().ReplicaSets(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(rs.Spec.Template.Spec.Volumes)

	case apis.KindStatefulSet:
		ss, err := opt.kubeClient.AppsV1().StatefulSets(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCsForStatefulset(ss.Spec.VolumeClaimTemplates, ss, replicas)

	case apis.KindPersistentVolumeClaim:
		pvcList = []string{targetRef.Name}
	}
	return pvcList, nil
}

func (opt *VSoption) getVolumeSnapshotDefinition(backupTarget *api_v1beta1.BackupTarget, namespace string, pvcName string, timestamp string) (volumeSnapshot vs.VolumeSnapshot) {
	return vs.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", pvcName, timestamp),
			Namespace: namespace,
		},
		Spec: vs.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &backupTarget.VolumeSnapshotClassName,
			Source: vs.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvcName,
			},
		},
	}
}

func getPVCs(volList []corev1.Volume) []string {
	pvcList := make([]string, 0)
	for _, vol := range volList {
		if vol.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, vol.PersistentVolumeClaim.ClaimName)
		}
	}
	return pvcList
}

func getPVCsForStatefulset(volList []corev1.PersistentVolumeClaim, ss *appsv1.StatefulSet, replicas *int32) []string {
	pvcList := make([]string, 0)
	var rep *int32
	if replicas != nil {
		rep = replicas
	} else {
		rep = ss.Spec.Replicas
	}
	for i := int32(0); i < *rep; i++ {
		podName := fmt.Sprintf("%v-%v", ss.Name, i)
		for _, vol := range volList {
			pvcList = append(pvcList, fmt.Sprintf("%v-%v", vol.Name, podName))
		}
	}
	return pvcList

}

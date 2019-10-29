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
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/pkg/volumesnapshot"

	"github.com/appscode/go/log"
	vs "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
)

type VSoption struct {
	backupsession  string
	restoresession string
	namespace      string
	kubeClient     kubernetes.Interface
	stashClient    cs.Interface
	snapshotClient vs_cs.Interface
	metrics        restic.MetricsOptions
}

func NewCmdCreateVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
			metrics: restic.MetricsOptions{
				Enabled: true,
				JobName: "stash-volumesnapshotter",
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
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = vs_cs.NewForConfigOrDie(config)

			backupOutput, err := opt.createVolumeSnapshot()
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
			return statOpt.UpdatePostBackupStatus(backupOutput)
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.backupsession, "backupsession", "", "Name of the respective BackupSession object")
	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	return cmd
}

func (opt *VSoption) createVolumeSnapshot() (*restic.BackupOutput, error) {
	// Start clock to measure total session duration
	startTime := time.Now()
	backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(opt.backupsession, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	backupConfig, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if backupConfig.Spec.Target == nil {
		return nil, fmt.Errorf("no target has been specified for BackupConfiguration %s/%s", backupConfig.Namespace, backupConfig.Name)
	}

	pvcNames, err := opt.getTargetPVCNames(backupConfig.Spec.Target.Ref, backupConfig.Spec.Target.Replicas)
	if err != nil {
		return nil, err
	}

	vsMeta := []metav1.ObjectMeta{}

	// create VolumeSnapshots
	for _, pvcName := range pvcNames {
		// use timestamp suffix of BackupSession name as suffix of the VolumeSnapshots name
		parts := strings.Split(backupSession.Name, "-")
		volumeSnapshot := opt.getVolumeSnapshotDefinition(backupConfig, pvcName, parts[len(parts)-1])
		snapshot, err := opt.snapshotClient.SnapshotV1alpha1().VolumeSnapshots(opt.namespace).Create(&volumeSnapshot)
		if err != nil {
			return nil, err
		}
		vsMeta = append(vsMeta, snapshot.ObjectMeta)
	}

	// now wait for all the VolumeSnapshots are completed (ready to to use)
	backupOutput := &restic.BackupOutput{}
	for i, pvcName := range pvcNames {
		// wait until this VolumeSnapshot is ready to use
		err = util.WaitUntilVolumeSnapshotReady(opt.snapshotClient, vsMeta[i])
		if err != nil {
			backupOutput.HostBackupStats = append(backupOutput.HostBackupStats, api_v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    api_v1beta1.HostBackupFailed,
				Error:    err.Error(),
			})
		} else {
			backupOutput.HostBackupStats = append(backupOutput.HostBackupStats, api_v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    api_v1beta1.HostBackupSucceeded,
				Duration: time.Since(startTime).String(),
			})
		}

	}

	err = volumesnapshot.CleanupSnapshots(backupConfig.Spec.RetentionPolicy, backupOutput.HostBackupStats, backupSession.Namespace, opt.snapshotClient)
	if err != nil {
		return nil, err
	}

	return backupOutput, nil
}

func (opt *VSoption) getTargetPVCNames(targetRef api_v1beta1.TargetRef, replicas *int32) ([]string, error) {
	var pvcList []string

	switch targetRef.Kind {
	case apis.KindDeployment:
		deployment, err := opt.kubeClient.AppsV1().Deployments(opt.namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(deployment.Spec.Template.Spec.Volumes)

	case apis.KindDaemonSet:
		daemon, err := opt.kubeClient.AppsV1().DaemonSets(opt.namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(daemon.Spec.Template.Spec.Volumes)

	case apis.KindReplicationController:
		rc, err := opt.kubeClient.CoreV1().ReplicationControllers(opt.namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(rc.Spec.Template.Spec.Volumes)

	case apis.KindReplicaSet:
		rs, err := opt.kubeClient.AppsV1().ReplicaSets(opt.namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCs(rs.Spec.Template.Spec.Volumes)

	case apis.KindStatefulSet:
		ss, err := opt.kubeClient.AppsV1().StatefulSets(opt.namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pvcList = getPVCsForStatefulset(ss.Spec.VolumeClaimTemplates, ss, replicas)

	case apis.KindPersistentVolumeClaim:
		pvcList = []string{targetRef.Name}
	}
	return pvcList, nil
}

func (opt *VSoption) getVolumeSnapshotDefinition(backupConfiguration *v1beta1.BackupConfiguration, pvcName string, timestamp string) (volumeSnapshot vs.VolumeSnapshot) {
	return vs.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", pvcName, timestamp),
			Namespace: backupConfiguration.Namespace,
		},
		Spec: vs.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &backupConfiguration.Spec.Target.VolumeSnapshotClassName,
			Source: &corev1.TypedLocalObjectReference{
				Kind: apis.KindPersistentVolumeClaim,
				Name: pvcName,
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

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

package volumesnapshot

import (
	"fmt"
	"testing"
	"time"

	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/appscode/go/strings"
	type_util "github.com/appscode/go/types"
	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	vsfake "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type snapInfo struct {
	name         string
	creationTime string
	pvcName      string
}

type testInfo struct {
	description       string
	policy            v1alpha1.RetentionPolicy
	hostBackupStats   []v1beta1.HostBackupStats
	expectedSnapshots []string
}

const testNamespace = "vs-retention-policy-test"

func TestCleanupSnapshots(t *testing.T) {

	snapMeta := []snapInfo{
		{name: "snap-1", creationTime: "2019-12-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-2", creationTime: "2019-11-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-3", creationTime: "2019-10-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-4", creationTime: "2019-10-10T05:30:07Z", pvcName: "pvc-1"},
		{name: "snap-5", creationTime: "2019-10-10T05:15:07Z", pvcName: "pvc-1"},
		{name: "snap-6", creationTime: "2018-10-11T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-7", creationTime: "2018-10-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-8", creationTime: "2017-12-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-9", creationTime: "2017-10-10T05:36:07Z", pvcName: "pvc-1"},
		{name: "snap-10", creationTime: "2019-12-10T05:36:07Z", pvcName: "pvc-2"},
		{name: "snap-11", creationTime: "2019-10-10T05:36:07Z", pvcName: "pvc-2"},
		{name: "snap-12", creationTime: "2019-10-10T05:36:07Z", pvcName: "pvc-2"},
		{name: "snap-13", creationTime: "2019-10-09T05:30:07Z", pvcName: "pvc-2"},
	}

	testCases := []testInfo{
		{
			description:       "No Policy",
			policy:            v1alpha1.RetentionPolicy{},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-4", "snap-5", "snap-6", "snap-7", "snap-8", "snap-9", "snap-10", "snap-11", "snap-12", "snap-13"}, // should keep all snapshots
		},
		{
			description:       "KeepLast",
			policy:            v1alpha1.RetentionPolicy{KeepLast: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-10", "snap-11", "snap-12"}, // should keep last 3 snapshots of claim 1 and last 3 snapshots of claim 2
		},
		{
			description:       "KeepHourly",
			policy:            v1alpha1.RetentionPolicy{KeepHourly: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-10", "snap-11", "snap-13"},
		},
		{
			description:       "KeepDaily",
			policy:            v1alpha1.RetentionPolicy{KeepDaily: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-10", "snap-11", "snap-13"},
		},
		{
			description:       "KeepMonthly",
			policy:            v1alpha1.RetentionPolicy{KeepMonthly: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-10", "snap-11"},
		},
		{
			description:       "KeepYearly",
			policy:            v1alpha1.RetentionPolicy{KeepYearly: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-6", "snap-8", "snap-10"},
		},
		{
			description:       "KeepLast & KeepDaily",
			policy:            v1alpha1.RetentionPolicy{KeepLast: 3, KeepDaily: 3},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-10", "snap-11", "snap-12", "snap-13"},
		},
		{
			description:       "KeepWeekly & KeepMonthly",
			policy:            v1alpha1.RetentionPolicy{KeepWeekly: 2, KeepMonthly: 2},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-10", "snap-11"},
		},
		{
			description:       "KeepWeekly & KeepMonthly & KeepYearly",
			policy:            v1alpha1.RetentionPolicy{KeepWeekly: 2, KeepMonthly: 3, KeepYearly: 4},
			hostBackupStats:   []v1beta1.HostBackupStats{{Hostname: "pvc-1"}, {Hostname: "pvc-2"}},
			expectedSnapshots: []string{"snap-1", "snap-2", "snap-3", "snap-6", "snap-8", "snap-10", "snap-11"},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			volumeSnasphots, err := getVolumeSnapshots(snapMeta)
			if err != nil {
				t.Errorf("Failed to generate VolumeSnasphots. Reason: %v", err)
				return
			}
			vsClient := vsfake.NewSimpleClientset(volumeSnasphots...)
			err = CleanupSnapshots(test.policy, test.hostBackupStats, testNamespace, vsClient)
			if err != nil {
				t.Errorf("Failed to cleanup VolumeSnapshots. Reason: %v", err)
				return
			}
			vsList, err := vsClient.SnapshotV1beta1().VolumeSnapshots(testNamespace).List(metav1.ListOptions{})
			if err != nil {
				t.Errorf("Failed to list remaining VolumeSnapshots. Reason: %v", err)
				return
			}
			if len(test.expectedSnapshots) != len(vsList.Items) {
				var remainingSnapshots []string
				for i := range vsList.Items {
					remainingSnapshots = append(remainingSnapshots, vsList.Items[i].Name)
				}

				t.Errorf("Remaining VolumeSnapshot number did not match with expected number."+
					"\nExpected: %d"+
					"\nFound: %d"+
					"\nExpected Snapshots: %q"+
					"\nRemaining Snapshots: %q", len(test.expectedSnapshots), len(vsList.Items), test.expectedSnapshots, remainingSnapshots)
				return
			}

			for _, vs := range vsList.Items {
				if !strings.Contains(test.expectedSnapshots, vs.Name) {
					t.Errorf("VolumeSnapshot %s should be deleted according to retention-policy: %s.", vs.Name, test.description)
				}
			}
		})
	}

}

func getVolumeSnapshots(snapMetas []snapInfo) ([]runtime.Object, error) {
	snapshots := make([]runtime.Object, 0)
	for i := range snapMetas {
		snapshot, err := newSnapshot(snapMetas[i])
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func newSnapshot(snapMeta snapInfo) (*crdv1.VolumeSnapshot, error) {
	creationTimestamp, err := time.Parse(time.RFC3339, snapMeta.creationTime)
	if err != nil {
		return nil, err
	}

	snapshotContentName := fmt.Sprintf("snapshot-content-%s", snapMeta.name)
	return &crdv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:              snapMeta.name,
			Namespace:         testNamespace,
			UID:               types.UID(snapMeta.name),
			ResourceVersion:   "1",
			SelfLink:          "/apis/snapshot.storage.k8s.io/v1alpha1/namespaces/" + testNamespace + "/volumesnapshots/" + snapMeta.name,
			CreationTimestamp: metav1.Time{Time: creationTimestamp},
		},
		Spec: crdv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: type_util.StringP("standard"),
			Source: crdv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &snapMeta.pvcName,
				VolumeSnapshotContentName: &snapshotContentName,
			},
		},
		Status: &crdv1.VolumeSnapshotStatus{
			ReadyToUse: type_util.TrueP(),
			Error:      nil,
		},
	}, nil
}

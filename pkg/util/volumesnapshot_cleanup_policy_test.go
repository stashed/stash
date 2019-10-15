package util

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/runtime"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vsfake "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/fake"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

const (
	testNamespace = "demo"
)

var (
	ClassName = "standard"
)

func creationTime(t string) time.Time {
	tm := parseTime(t)
	return tm
}

func parseTime(t string) time.Time {
	tm, err := time.Parse(time.RFC3339, t)
	if err != nil {
		panic(err)
	}
	return tm
}

func TestCleanupSnapshots(t *testing.T) {

	//var policy = []v1alpha1.RetentionPolicy{
	//	{Name: "No Policy"},
	//	{Name: "KeepLast", KeepLast: 3},
	//	{Name: "KeepHourly", KeepHourly: 3},
	//	{Name: "KeepDaily", KeepDaily: 3},
	//	{Name: "KeepWeakly", KeepWeekly: 3},
	//	{Name: "KeepMonthly", KeepMonthly: 3},
	//	{Name: "KeepYearly", KeepYearly: 3},
	//	{Name: "KeepLast & KeepHourly", KeepLast: 3, KeepHourly: 3},
	//	{Name: "KeepLast & KeepDaily", KeepLast: 3, KeepDaily: 3},
	//	{Name: "KeepLast & KeepDaily", KeepLast: 2, KeepDaily: 3},
	//	{Name: "KeepLast & KeepWeekly & KeepMonthly", KeepHourly: 2, KeepWeekly: 2, KeepMonthly: 2},
	//	{Name: "KeepLast & KeepMonthly & KeepYearly", KeepLast: 3, KeepMonthly: 2, KeepYearly: 2},
	//	{Name: "KeepWeekly & KeepMonthly & KeepYearly", KeepWeekly: 2, KeepMonthly: 2, KeepYearly: 3},
	//}

	var volumeSnaps = []*crdv1.VolumeSnapshot{
		newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
		newSnapshot("snap2", ClassName, "snapcontent-snapuid2", "snapuid2", "claim1", true, nil, creationTime("2018-10-10T05:36:07Z")),
		newSnapshot("snap3", ClassName, "snapcontent-snapuid3", "snapuid3", "claim1", true, nil, creationTime("2017-10-10T05:36:07Z")),
		newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
		newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
		newSnapshot("snap6", ClassName, "snapcontent-snapuid6", "snapuid6", "claim1", true, nil, creationTime("2019-10-10T05:30:07Z")),
		newSnapshot("snap7", ClassName, "snapcontent-snapuid7", "snapuid7", "claim1", true, nil, creationTime("2019-10-10T05:15:07Z")),
		newSnapshot("snap8", ClassName, "snapcontent-snapuid8", "snapuid8", "claim1", true, nil, creationTime("2018-10-11T05:36:07Z")),
		newSnapshot("snap9", ClassName, "snapcontent-snapuid9", "snapuid9", "claim1", true, nil, creationTime("2017-12-10T05:36:07Z")),
		newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
		newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
	}
	var objs []runtime.Object
	for _, vs := range volumeSnaps {
		objs = append(objs, vs)
	}

	var tests = []struct {
		description     string
		namespace       string
		policy          v1alpha1.RetentionPolicy
		hostBackupStats []v1beta1.HostBackupStats
		expected        []*crdv1.VolumeSnapshot
		objects         []runtime.Object
	}{
		{
			description:     "No Policy",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}},
			expected:        volumeSnaps,
			objects:         objs,
		},
		{
			description:     "KeepLast",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepLast: 3},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim1"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepHourly",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepHourly: 3},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepDaily",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepDaily: 3},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepYearly",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepYearly: 3},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap8", ClassName, "snapcontent-snapuid8", "snapuid8", "claim1", true, nil, creationTime("2018-10-11T05:36:07Z")),
				newSnapshot("snap9", ClassName, "snapcontent-snapuid9", "snapuid9", "claim1", true, nil, creationTime("2017-12-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepLast & KeepDaily",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepLast: 3, KeepDaily: 3},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepWeekly & KeepMonthly",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepWeekly: 2, KeepMonthly: 2},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
		{
			description:     "KeepWeekly & KeepMonthly & KeepYearly",
			namespace:       testNamespace,
			policy:          v1alpha1.RetentionPolicy{KeepWeekly: 2, KeepMonthly: 3, KeepYearly: 4},
			hostBackupStats: []v1beta1.HostBackupStats{{Hostname: "claim1"}, {Hostname: "claim2"}},
			expected: []*crdv1.VolumeSnapshot{
				newSnapshot("snap1", ClassName, "snapcontent-snapuid1", "snapuid1", "claim1", true, nil, creationTime("2019-10-10T05:36:07Z")),
				newSnapshot("snap4", ClassName, "snapcontent-snapuid4", "snapuid4", "claim1", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap5", ClassName, "snapcontent-snapuid5", "snapuid5", "claim1", true, nil, creationTime("2019-12-10T05:36:07Z")),
				newSnapshot("snap8", ClassName, "snapcontent-snapuid8", "snapuid8", "claim1", true, nil, creationTime("2018-10-11T05:36:07Z")),
				newSnapshot("snap9", ClassName, "snapcontent-snapuid9", "snapuid9", "claim1", true, nil, creationTime("2017-12-10T05:36:07Z")),
				newSnapshot("snap10", ClassName, "snapcontent-snapuid4", "snapuid4", "claim2", true, nil, creationTime("2019-11-10T05:36:07Z")),
				newSnapshot("snap11", ClassName, "snapcontent-snapuid5", "snapuid5", "claim2", true, nil, creationTime("2019-12-10T05:36:07Z")),
			},
			objects: objs,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			vsClient := vsfake.NewSimpleClientset(test.objects...)
			err := CleanupSnapshots(test.policy, test.hostBackupStats, test.namespace, vsClient)
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
				return
			}
			vsList, err := vsClient.SnapshotV1alpha1().VolumeSnapshots(test.namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
				return
			}
			if len(vsList.Items) != len(test.expected) {
				t.Errorf(cmp.Diff(len(vsList.Items), len(test.expected)))
				return
			}

			for i, vs := range vsList.Items {
				if !cmp.Equal(vs, *test.expected[i]) {
					t.Errorf(cmp.Diff(vs, *test.expected[i]))
					return
				}
			}
		})
	}

}

func newSnapshot(name, className, boundToContent, snapshotUID, claimName string, ready bool, err *storagev1beta1.VolumeError, creationTime time.Time) *crdv1.VolumeSnapshot {
	snapshot := crdv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       testNamespace,
			UID:             types.UID(snapshotUID),
			ResourceVersion: "1",
			SelfLink:        "/apis/snapshot.storage.k8s.io/v1alpha1/namespaces/" + testNamespace + "/volumesnapshots/" + name,
		},
		Spec: crdv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &className,
			SnapshotContentName:     boundToContent,
		},
		Status: crdv1.VolumeSnapshotStatus{
			ReadyToUse: ready,
			Error:      err,
		},
	}
	snapshot.Spec.Source = &v1.TypedLocalObjectReference{
		Name: claimName,
		Kind: "PersistentVolumeClaim",
	}
	snapshot.SetCreationTimestamp(metav1.Time{Time: creationTime})
	return &snapshot
}

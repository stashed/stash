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
	"sort"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/appscode/go/log"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Some of the code of this file has been copied from restic/restic repository.
// ref: https://github.com/restic/restic/blob/604b18aa7426148a55f76307ca729e829ff6b61d/internal/restic/snapshot_policy.go#L152:6

// isPolicyEmpty returns true if no policy has been configured (all values zero).
func isPolicyEmpty(policy v1alpha1.RetentionPolicy) bool {
	if policy.KeepLast > 0 ||
		policy.KeepHourly > 0 ||
		policy.KeepDaily > 0 ||
		policy.KeepMonthly > 0 ||
		policy.KeepWeekly > 0 ||
		policy.KeepYearly > 0 {
		return true
	}
	return false
}

// ymdh returns an integer in the form YYYYMMDDHH.
func ymdh(d time.Time, _ int) int {
	return d.Year()*1000000 + int(d.Month())*10000 + d.Day()*100 + d.Hour()
}

// ymd returns an integer in the form YYYYMMDD.
func ymd(d time.Time, _ int) int {
	return d.Year()*10000 + int(d.Month())*100 + d.Day()
}

// yw returns an integer in the form YYYYWW, where WW is the week number.
func yw(d time.Time, _ int) int {
	year, week := d.ISOWeek()
	return year*100 + week
}

// ym returns an integer in the form YYYYMM.
func ym(d time.Time, _ int) int {
	return d.Year()*100 + int(d.Month())
}

// y returns the year of d.
func y(d time.Time, _ int) int {
	return d.Year()
}

// always returns a unique number for d.
func always(d time.Time, nr int) int {
	return nr
}

type VolumeSnapshot struct {
	VolumeSnap vs_api.VolumeSnapshot
}

type VolumeSnapshots []VolumeSnapshot

func (vs VolumeSnapshots) Len() int {
	return len(vs)
}
func (vs VolumeSnapshots) Less(i, j int) bool {
	return vs[i].VolumeSnap.CreationTimestamp.Time.After(vs[j].VolumeSnap.CreationTimestamp.Time)
}
func (vs VolumeSnapshots) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// ApplyRetentionPolicy do the following steps:
// 1. sorts all the VolumeSnapshot according to CreationTimeStamp.
// 2. then list that are to be kept and removed according to the policy.
// 3. remove VolumeSnapshot that are not necessary according to RetentionPolicy
func applyRetentionPolicy(policy v1alpha1.RetentionPolicy, volumeSnapshots VolumeSnapshots, namespace string, vsClient vs_cs.Interface) error {

	// sorts the VolumeSnapshots according to CreationTimeStamp
	sort.Sort(VolumeSnapshots(volumeSnapshots))

	if !isPolicyEmpty(policy) {
		return nil
	}

	var buckets = [6]struct {
		Count     int64
		LastAdded func(d time.Time, nr int) int
		Last      int
	}{
		{policy.KeepLast, always, -1},
		{policy.KeepHourly, ymdh, -1},
		{policy.KeepDaily, ymd, -1},
		{policy.KeepWeekly, yw, -1},
		{policy.KeepMonthly, ym, -1},
		{policy.KeepYearly, y, -1},
	}

	var kept, removed VolumeSnapshots
	for nr, vs := range volumeSnapshots {
		var keepSnap bool
		// keep VolumeSnapshot that are matched with the policy
		for i, b := range buckets {
			if b.Count > 0 {
				val := b.LastAdded(vs.VolumeSnap.CreationTimestamp.Time, nr)
				if val != b.Last {
					keepSnap = true
					buckets[i].Last = val
					buckets[i].Count--
				}
			}
		}

		if keepSnap {
			kept = append(kept, vs)
		} else {
			removed = append(removed, vs)
		}
	}

	for _, vs := range removed {
		err := vsClient.SnapshotV1beta1().VolumeSnapshots(namespace).Delete(vs.VolumeSnap.Name, &v1.DeleteOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				return nil
			}
			return err
		}
	}

	log.Infof("VolumeSnapshot kept: %d removed: %d", len(kept), len(removed))
	return nil
}

func CleanupSnapshots(policy v1alpha1.RetentionPolicy, hostBackupStats []v1beta1.HostBackupStats, namespace string, vsClient vs_cs.Interface) error {
	vsList, err := vsClient.SnapshotV1beta1().VolumeSnapshots(namespace).List(v1.ListOptions{})
	if err != nil {
		if kerr.IsNotFound(err) || len(vsList.Items) == 0 {
			return nil
		}
		return err
	}
	// filter VolumeSnapshots according to PVC source
	// then apply RetentionPolicy rule
	for _, host := range hostBackupStats {
		var volumeSnapshots VolumeSnapshots
		for _, vs := range vsList.Items {
			var src string
			if vs.Spec.Source.PersistentVolumeClaimName != nil {
				src = *vs.Spec.Source.PersistentVolumeClaimName
			}
			if host.Hostname == src {
				volumeSnapshots = append(volumeSnapshots, VolumeSnapshot{VolumeSnap: vs})
			}
		}
		err := applyRetentionPolicy(policy, volumeSnapshots, namespace, vsClient)
		if err != nil {
			return err
		}
	}

	return nil
}

package volumesnapshot

import (
	"sort"
	"time"

	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

// Empty returns true iff no policy has been configured (all values zero).
func IsPolicyEmpty(policy v1alpha1.RetentionPolicy) bool {
	if policy.KeepLast > 0 || policy.KeepHourly > 0 || policy.KeepDaily > 0 || policy.KeepMonthly > 0 || policy.KeepWeekly > 0 || policy.KeepYearly > 0 {
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
	vs vs_api.VolumeSnapshot
}

type VolumeSnapshots []VolumeSnapshot

func (vs VolumeSnapshots) Len() int {
	return len(vs)
}
func (vs VolumeSnapshots) Less(i, j int) bool {
	return vs[i].vs.CreationTimestamp.Time.After(vs[j].vs.CreationTimestamp.Time)
}
func (vs VolumeSnapshots) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// ApplyRetentionPolicy do the following steps:
// 1. sorts all the VolumeSnapshot according to CreationTimeStamp.
// 2. then list that are to be kept and removed according to the policy.
// 3. then remove those VolumeSnapshot according to policy that are not necessary.
func ApplyRetentionPolicy(policy v1alpha1.RetentionPolicy, namespace string, vsClient vs_cs.Interface) error {
	vsList, err := vsClient.SnapshotV1alpha1().VolumeSnapshots(namespace).List(v1.ListOptions{})
	if err != nil {
		if kerr.IsNotFound(err) || len(vsList.Items) == 0 {
			return nil
		}
		return err
	}

	var volumeSnapshots VolumeSnapshots

	for _, vs := range vsList.Items {
		volumeSnapshots = append(volumeSnapshots, VolumeSnapshot{vs: vs})
	}

	// sorts the VolumeSnapshots according to CreationTimeStamp
	sort.Sort(VolumeSnapshots(volumeSnapshots))

	if !IsPolicyEmpty(policy) {
		return nil
	}

	var buckets = [6]struct {
		Count  int
		bucker func(d time.Time, nr int) int
		Last   int
	}{
		{policy.KeepLast, always, -1},
		{policy.KeepHourly, ymdh, -1},
		{policy.KeepDaily, ymd, -1},
		{policy.KeepWeekly, yw, -1},
		{policy.KeepMonthly, ym, -1},
		{policy.KeepYearly, y, -1},
	}

	var keep, remove VolumeSnapshots
	for nr, vs := range volumeSnapshots {
		var keepSnap bool
		for i, b := range buckets {
			if b.Count > 0 {
				val := b.bucker(vs.vs.CreationTimestamp.Time, nr)
				if val != b.Last {
					keepSnap = true
					buckets[i].Last = val
					buckets[i].Count--
				}
			}
		}
		if keepSnap {
			keep = append(keep, vs)
		} else {
			remove = append(remove, vs)
		}

	}

	for _, vs := range remove {
		err := vsClient.SnapshotV1alpha1().VolumeSnapshots(namespace).Delete(vs.vs.Name, &v1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

package volumesnapshot

import (
	"sort"
	"time"

	"github.com/appscode/go/log"

	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
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

type volumesnapshot struct {
	vs   vs_api.VolumeSnapshot
}


func (a volumesnapshot) Len() int           { return len(a) }
func (a volumesnapshot) Less(i, j int) bool { return a[i].vs.CreationTimestamp.Time < a[j].vs.CreationTimestamp.Time}
func (a volumesnapshot) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }


func ApplyRetentionPolicy(policy v1alpha1.RetentionPolicy, namespace string, vsClient vs_cs.Interface) error {
	vsList, err := vsClient.SnapshotV1alpha1().VolumeSnapshots(namespace).List(v1.ListOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	var vss []volumesnapshot

	for _, vs := range vsList.Items {
		vss = append(vss, volumesnapshot{vs:vs})
	}

	if len(vsList.Items) == 0 {
		return nil
	}

	sort.Sort(vsshot())

	var buckets = [6]struct {
		Count  int
		bucker func(d time.Time, nr int) int
		Last   int
		reason string
	}{
		{policy.KeepLast, always, -1, "last snapshot"},
		{policy.KeepHourly, ymdh, -1, "hourly snapshot"},
		{policy.KeepDaily, ymd, -1, "daily snapshot"},
		{policy.KeepWeekly, yw, -1, "weekly snapshot"},
		{policy.KeepMonthly, ym, -1, "monthly snapshot"},
		{policy.KeepYearly, y, -1, "yearly snapshot"},
	}

	for nr, vs := range vsList.Items {
		var keepSnap bool
		var keepSnapReasons []string

		for i, b := range buckets {
			if b.Count > 0 {
				val := b.bucker(vs.CreationTimestamp.Time, nr)
				if val != b.Last {
					log.Infoln("keep %v %v, bucker %v, val %v\n", vs.CreationTimestamp.Time, string(vs.UID), i, val)
					keepSnap = true
					buckets[i].Last = val
					buckets[i].Count--
					keepSnapReasons = append(keepSnapReasons, b.reason)
				}
			}
		}

	}

}

package cli

import (
	"strconv"
	"time"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	shell "github.com/codeskyblue/go-sh"
)

const (
	Exe = "/bin/restic"
)

type ResticWrapper struct {
	sh         *shell.Session
	scratchDir string
	hostname   string
}

func New(scratchDir, hostname string) *ResticWrapper {
	ctrl := &ResticWrapper{
		sh:         shell.NewSession(),
		scratchDir: scratchDir,
		hostname:   hostname,
	}
	ctrl.sh.SetDir(scratchDir)
	ctrl.sh.ShowCMD = true
	return ctrl
}

type Snapshot struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	UID      int       `json:"uid"`
	Gid      int       `json:"gid"`
}

func (w *ResticWrapper) ListSnapshots() ([]Snapshot, error) {
	result := make([]Snapshot, 0)
	err := w.sh.Command(Exe, "snapshots", "--json").UnmarshalJSON(&result)
	return result, err
}

func (w *ResticWrapper) InitRepositoryIfAbsent() error {
	if err := w.sh.Command(Exe, "snapshots", "--json").Run(); err != nil {
		return w.sh.Command(Exe, "init").Run()
	}
	return nil
}

func (w *ResticWrapper) Backup(resource *api.Restic, fg api.FileGroup) error {
	args := []interface{}{"backup", fg.Path, "--force"}
	if w.hostname != "" {
		args = append(args, "--hostname")
		args = append(args, w.hostname)
	}
	// add tags if any
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	return w.sh.Command(Exe, args...).Run()
}

func (w *ResticWrapper) Forget(resource *api.Restic, fg api.FileGroup) error {
	args := []interface{}{"forget"}
	if fg.RetentionPolicy.KeepLast > 0 {
		args = append(args, string(api.KeepLast))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepLast))
	}
	if fg.RetentionPolicy.KeepHourly > 0 {
		args = append(args, string(api.KeepHourly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepHourly))
	}
	if fg.RetentionPolicy.KeepDaily > 0 {
		args = append(args, string(api.KeepDaily))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepDaily))
	}
	if fg.RetentionPolicy.KeepWeekly > 0 {
		args = append(args, string(api.KeepWeekly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepWeekly))
	}
	if fg.RetentionPolicy.KeepMonthly > 0 {
		args = append(args, string(api.KeepMonthly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepMonthly))
	}
	if fg.RetentionPolicy.KeepYearly > 0 {
		args = append(args, string(api.KeepYearly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepYearly))
	}
	for _, tag := range fg.RetentionPolicy.KeepTags {
		args = append(args, string(api.KeepTag))
		args = append(args, tag)
	}
	if fg.RetentionPolicy.Prune {
		args = append(args, "--prune")
	}
	if fg.RetentionPolicy.DryRun {
		args = append(args, "--dry-run")
	}
	if len(args) > 1 {
		err := w.sh.Command(Exe, args...).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *ResticWrapper) Restore(recovery *api.Recovery) error {
	args := []interface{}{"restore"}
	if len(recovery.Spec.SnapshotID) != 0 {
		args = append(args, recovery.Spec.SnapshotID)
	} else {
		args = append(args, "latest")
	}
	if len(recovery.Spec.Host) != 0 {
		args = append(args, "--host")
		args = append(args, recovery.Spec.Host)
	}
	if len(recovery.Spec.Path) != 0 {
		args = append(args, "--path")
		args = append(args, recovery.Spec.Path)
	}
	args = append(args, "--target")
	args = append(args, recovery.Spec.VolumeMounts[0].MountPath)
	return w.sh.Command(Exe, args...).Run()
}

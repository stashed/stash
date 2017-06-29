package cli

import (
	"strconv"
	"time"

	sapi "github.com/appscode/stash/api"
	shell "github.com/codeskyblue/go-sh"
)

const (
	Exe = "/bin/restic"
)

type ResticWrapper struct {
	sh         *shell.Session
	scratchDir string
}

func New(scratchDir string) *ResticWrapper {
	ctrl := &ResticWrapper{
		sh:         shell.NewSession(),
		scratchDir: scratchDir,
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

func (w *ResticWrapper) Backup(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"backup", fg.Path, "--force"}
	// add tags if any
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	return w.sh.Command(Exe, args...).Run()
}

func (w *ResticWrapper) Forget(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"forget"}
	if fg.RetentionPolicy.KeepLast > 0 {
		args = append(args, string(sapi.KeepLast))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepLast))
	}
	if fg.RetentionPolicy.KeepHourly > 0 {
		args = append(args, string(sapi.KeepHourly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepHourly))
	}
	if fg.RetentionPolicy.KeepDaily > 0 {
		args = append(args, string(sapi.KeepDaily))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepDaily))
	}
	if fg.RetentionPolicy.KeepWeekly > 0 {
		args = append(args, string(sapi.KeepWeekly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepWeekly))
	}
	if fg.RetentionPolicy.KeepMonthly > 0 {
		args = append(args, string(sapi.KeepMonthly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepMonthly))
	}
	if fg.RetentionPolicy.KeepYearly > 0 {
		args = append(args, string(sapi.KeepYearly))
		args = append(args, strconv.Itoa(fg.RetentionPolicy.KeepYearly))
	}
	for _, tag := range fg.RetentionPolicy.KeepTags {
		args = append(args, string(sapi.KeepTag))
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

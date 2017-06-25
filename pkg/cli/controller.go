package cli

import (
	sapi "github.com/appscode/stash/api"
	shell "github.com/codeskyblue/go-sh"
)

const (
	Exe = "/restic"
)

type resticWrapper struct {
	sh             *shell.Session
	scratchDir     string
	prefixHostname bool
}

func New(scratchDir string, prefixHostname bool) *resticWrapper {
	ctrl := &resticWrapper{
		sh:             shell.NewSession(),
		scratchDir:     scratchDir,
		prefixHostname: prefixHostname,
	}
	ctrl.sh.SetDir(scratchDir)
	ctrl.sh.ShowCMD = true
	return ctrl
}

func (w *resticWrapper) ListSnapshots() error {
	err := w.sh.Command(Exe, "snapshots", "--json").Run()
	return nil
}

func (w *resticWrapper) InitRepositoryIfAbsent() error {
	if err := w.sh.Command(Exe, "snapshots", "--json").Run(); err != nil {
		err = w.sh.Command(Exe, "init").Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *resticWrapper) Backup(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"backup", fg.Path, "--force"}
	// add tags if any
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	return w.sh.Command(Exe, args...).Run()
}

func (w *resticWrapper) Forget(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"forget"}
	if fg.RetentionPolicy.KeepLastSnapshots > 0 {
		args = append(args, sapi.KeepLast)
		args = append(args, fg.RetentionPolicy.KeepLastSnapshots)
	}
	if fg.RetentionPolicy.KeepHourlySnapshots > 0 {
		args = append(args, sapi.KeepHourly)
		args = append(args, fg.RetentionPolicy.KeepHourlySnapshots)
	}
	if fg.RetentionPolicy.KeepDailySnapshots > 0 {
		args = append(args, sapi.KeepDaily)
		args = append(args, fg.RetentionPolicy.KeepDailySnapshots)
	}
	if fg.RetentionPolicy.KeepWeeklySnapshots > 0 {
		args = append(args, sapi.KeepWeekly)
		args = append(args, fg.RetentionPolicy.KeepWeeklySnapshots)
	}
	if fg.RetentionPolicy.KeepMonthlySnapshots > 0 {
		args = append(args, sapi.KeepMonthly)
		args = append(args, fg.RetentionPolicy.KeepMonthlySnapshots)
	}
	if fg.RetentionPolicy.KeepYearlySnapshots > 0 {
		args = append(args, sapi.KeepYearly)
		args = append(args, fg.RetentionPolicy.KeepYearlySnapshots)
	}
	for _, tag := range fg.RetentionPolicy.KeepTags {
		args = append(args, "--keep-tag")
		args = append(args, tag)
	}
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	err := w.sh.Command(Exe, args...).Run()
	if err != nil {
		return err
	}
	return nil
}

package restic

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/pkg/errors"
)

const (
	Exe = "/bin/restic_0.9.4"
)

type Snapshot struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	UID      int       `json:"uid"`
	Gid      int       `json:"gid"`
	Tags     []string  `json:"tags"`
}

func (w *ResticWrapper) listSnapshots(snapshotIDs []string) ([]Snapshot, error) {
	result := make([]Snapshot, 0)
	args := w.appendCacheDirFlag([]interface{}{"snapshots", "--json", "--quiet", "--no-lock"})
	args = w.appendCaCertFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}

	err := w.sh.Command(Exe, args...).UnmarshalJSON(&result)
	return result, err
}

func (w *ResticWrapper) deleteSnapshots(snapshotIDs []string) ([]byte, error) {
	args := w.appendCacheDirFlag([]interface{}{"forget", "--quiet", "--prune"})
	args = w.appendCaCertFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}

	return w.run(Exe, args)
}

func (w *ResticWrapper) initRepositoryIfAbsent() ([]byte, error) {
	log.Infoln("Ensuring restic repository in the backend")
	args := w.appendCacheDirFlag([]interface{}{"snapshots", "--json"})
	args = w.appendCaCertFlag(args)
	if _, err := w.run(Exe, args); err != nil {
		args = w.appendCacheDirFlag([]interface{}{"init"})
		args = w.appendCaCertFlag(args)

		return w.run(Exe, args)
	}
	return nil, nil
}

func (w *ResticWrapper) backup(path, host string, tags []string) ([]byte, error) {
	log.Infoln("Backing up target data")
	args := []interface{}{"backup", path}
	if host != "" {
		args = append(args, "--host")
		args = append(args, host)
	}
	// add tags if any
	for _, tag := range tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) cleanup(retentionPolicy v1alpha1.RetentionPolicy) ([]byte, error) {
	log.Infoln("Cleaning old snapshots according to retention policy")

	args := []interface{}{"forget"}

	if retentionPolicy.KeepLast > 0 {
		args = append(args, string(v1alpha1.KeepLast))
		args = append(args, strconv.Itoa(retentionPolicy.KeepLast))
	}
	if retentionPolicy.KeepHourly > 0 {
		args = append(args, string(v1alpha1.KeepHourly))
		args = append(args, strconv.Itoa(retentionPolicy.KeepHourly))
	}
	if retentionPolicy.KeepDaily > 0 {
		args = append(args, string(v1alpha1.KeepDaily))
		args = append(args, strconv.Itoa(retentionPolicy.KeepDaily))
	}
	if retentionPolicy.KeepWeekly > 0 {
		args = append(args, string(v1alpha1.KeepWeekly))
		args = append(args, strconv.Itoa(retentionPolicy.KeepWeekly))
	}
	if retentionPolicy.KeepMonthly > 0 {
		args = append(args, string(v1alpha1.KeepMonthly))
		args = append(args, strconv.Itoa(retentionPolicy.KeepMonthly))
	}
	if retentionPolicy.KeepYearly > 0 {
		args = append(args, string(v1alpha1.KeepYearly))
		args = append(args, strconv.Itoa(retentionPolicy.KeepYearly))
	}
	for _, tag := range retentionPolicy.KeepTags {
		args = append(args, string(v1alpha1.KeepTag))
		args = append(args, tag)
	}
	if retentionPolicy.Prune {
		args = append(args, "--prune")
	}
	if retentionPolicy.DryRun {
		args = append(args, "--dry-run")
	}

	if len(args) > 1 {
		args = w.appendCacheDirFlag(args)
		args = w.appendCaCertFlag(args)

		return w.run(Exe, args)
	}
	return nil, nil
}

func (w *ResticWrapper) restore(path, host, snapshotID string) ([]byte, error) {
	log.Infoln("Restoring backed up data")
	args := []interface{}{"restore"}
	if snapshotID != "" {
		args = append(args, snapshotID)
	} else {
		args = append(args, "latest")
	}
	args = append(args, "--path")
	args = append(args, path) // source-path specified in restic fileGroup
	args = append(args, "--host")
	args = append(args, host)

	// Remove last part from the path.
	// https://github.com/appscode/stash/issues/392
	args = append(args, "--target", "/")
	// args = append(args, filepath.Dir(path))

	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) check() ([]byte, error) {
	log.Infoln("Checking integrity of repository")
	args := w.appendCacheDirFlag([]interface{}{"check"})
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) stats() ([]byte, error) {
	log.Infoln("Reading repository status")
	args := w.appendCacheDirFlag([]interface{}{"stats"})
	args = append(args, "--mode=raw-data", "--quiet")
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) appendCacheDirFlag(args []interface{}) []interface{} {
	if w.config.EnableCache {
		cacheDir := filepath.Join(w.config.ScratchDir, "restic-cache")
		return append(args, "--cache-dir", cacheDir)
	}
	return append(args, "--no-cache")
}

func (w *ResticWrapper) appendCaCertFlag(args []interface{}) []interface{} {
	if w.config.CacertFile != "" {
		return append(args, "--cacert", w.config.CacertFile)
	}
	return args
}

func (w *ResticWrapper) run(cmd string, args []interface{}) ([]byte, error) {
	out, err := w.sh.Command(cmd, args...).Output()
	if err != nil {
		log.Errorf("Error running command '%s %s' output:\n%s", cmd, args, string(out))
		parts := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
		if len(parts) > 1 {
			parts = parts[len(parts)-1:]
			return nil, errors.New(parts[0])
		}
	}
	return out, err
}

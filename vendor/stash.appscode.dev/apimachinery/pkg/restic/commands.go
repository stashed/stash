/*
Copyright AppsCode Inc. and Contributors

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

package restic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	"github.com/armon/circbuf"
	"k8s.io/klog/v2"
	storage "kmodules.xyz/objectstore-api/api/v1"
)

const (
	ResticCMD = "/bin/restic"
	BashCMD   = "/bin/bash"
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

type backupParams struct {
	path     string
	host     string
	tags     []string
	excludes []string
	args     []string
}

type restoreParams struct {
	path        string
	host        string
	snapshotId  string
	destination string
	excludes    []string
	includes    []string
	args        []string
}

type keyParams struct {
	id   string
	user string
	host string
	file string
}

func (w *ResticWrapper) listSnapshots(snapshotIDs []string) ([]Snapshot, error) {
	result := make([]Snapshot, 0)
	args := w.appendCacheDirFlag([]any{"snapshots", "--json", "--quiet", "--no-lock"})
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}
	out, err := w.run(Command{Name: ResticCMD, Args: args})
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &result)
	return result, err
}

func (w *ResticWrapper) deleteSnapshots(snapshotIDs []string) ([]byte, error) {
	args := w.appendCacheDirFlag([]any{"forget", "--quiet", "--prune"})
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) repositoryExist() bool {
	klog.Infoln("Checking whether the backend repository exist or not....")
	args := w.appendCacheDirFlag([]any{"snapshots", "--json", "--no-lock"})
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	if _, err := w.run(Command{Name: ResticCMD, Args: args}); err == nil {
		return true
	}
	return false
}

func (w *ResticWrapper) initRepository() error {
	klog.Infoln("Initializing new restic repository in the backend....")
	if err := w.createLocalDir(); err != nil {
		return err
	}

	args := w.appendCacheDirFlag([]any{"init"})
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	_, err := w.run(Command{Name: ResticCMD, Args: args})
	return err
}

func (w *ResticWrapper) backup(params backupParams) ([]byte, error) {
	klog.Infoln("Backing up target data")
	args := []any{"backup", params.path, "--quiet", "--json"}
	if params.host != "" {
		args = append(args, "--host")
		args = append(args, params.host)
	}
	// add tags if any
	for _, tag := range params.tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	// add exclude patterns if there any
	for _, exclude := range params.excludes {
		args = append(args, "--exclude")
		args = append(args, exclude)
	}
	// add additional arguments passed by user to the backup process
	for i := range params.args {
		args = append(args, params.args[i])
	}
	args = w.appendCacheDirFlag(args)
	args = w.appendCleanupCacheFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) backupFromStdin(options BackupOptions) ([]byte, error) {
	klog.Infoln("Backing up stdin data")

	// first add StdinPipeCommands, then add restic command
	commands := options.StdinPipeCommands

	args := []any{"backup", "--stdin", "--quiet", "--json"}
	if options.StdinFileName != "" {
		args = append(args, "--stdin-filename")
		args = append(args, options.StdinFileName)
	}
	if options.Host != "" {
		args = append(args, "--host")
		args = append(args, options.Host)
	}
	args = w.appendCacheDirFlag(args)
	args = w.appendCleanupCacheFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)

	commands = append(commands, Command{Name: ResticCMD, Args: args})
	return w.run(commands...)
}

func (w *ResticWrapper) createLocalDir() error {
	if w.config.Provider == storage.ProviderLocal {
		return os.MkdirAll(w.config.Bucket, 0o755)
	}
	return nil
}

func (w *ResticWrapper) cleanup(retentionPolicy v1alpha1.RetentionPolicy, host string) ([]byte, error) {
	klog.Infoln("Cleaning old snapshots according to retention policy")

	out, err := w.tryCleanup(retentionPolicy, host)
	if err == nil || !strings.Contains(err.Error(), "unlock") {
		return out, err
	}
	// repo is locked, so unlock first
	klog.Warningln("repo found locked, so unlocking before pruning, err:", err.Error())
	if o2, e2 := w.unlock(); e2 != nil {
		return o2, e2
	}
	return w.tryCleanup(retentionPolicy, host)
}

func (w *ResticWrapper) tryCleanup(retentionPolicy v1alpha1.RetentionPolicy, host string) ([]byte, error) {
	args := []any{"forget", "--quiet", "--json"}

	if host != "" {
		args = append(args, "--host")
		args = append(args, host)
	}

	if retentionPolicy.KeepLast > 0 {
		args = append(args, string(v1alpha1.KeepLast))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepLast, 10))
	}
	if retentionPolicy.KeepHourly > 0 {
		args = append(args, string(v1alpha1.KeepHourly))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepHourly, 10))
	}
	if retentionPolicy.KeepDaily > 0 {
		args = append(args, string(v1alpha1.KeepDaily))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepDaily, 10))
	}
	if retentionPolicy.KeepWeekly > 0 {
		args = append(args, string(v1alpha1.KeepWeekly))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepWeekly, 10))
	}
	if retentionPolicy.KeepMonthly > 0 {
		args = append(args, string(v1alpha1.KeepMonthly))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepMonthly, 10))
	}
	if retentionPolicy.KeepYearly > 0 {
		args = append(args, string(v1alpha1.KeepYearly))
		args = append(args, strconv.FormatInt(retentionPolicy.KeepYearly, 10))
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
		args = w.appendInsecureTLSFlag(args)
		args = w.appendMaxConnectionsFlag(args)

		return w.run(Command{Name: ResticCMD, Args: args})
	}
	return nil, nil
}

func (w *ResticWrapper) restore(params restoreParams) ([]byte, error) {
	klog.Infoln("Restoring backed up data")

	args := []any{"restore"}
	if params.snapshotId != "" {
		args = append(args, params.snapshotId)
	} else {
		args = append(args, "latest")
	}
	if params.path != "" {
		args = append(args, "--path")
		args = append(args, params.path) // source-path specified in restic fileGroup
	}
	if params.host != "" {
		args = append(args, "--host")
		args = append(args, params.host)
	}

	if params.destination == "" {
		params.destination = "/" // restore in absolute path
	}
	args = append(args, "--target", params.destination)

	// add include patterns if there any
	for _, include := range params.includes {
		args = append(args, "--include")
		args = append(args, include)
	}
	// add exclude patterns if there any
	for _, exclude := range params.excludes {
		args = append(args, "--exclude")
		args = append(args, exclude)
	}
	// add additional arguments passed by user to the restore process
	for i := range params.args {
		args = append(args, params.args[i])
	}
	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

// Redis cluster directly calls the DumpOnce method
func (w *ResticWrapper) DumpOnce(dumpOptions DumpOptions) ([]byte, error) {
	klog.Infoln("Dumping backed up data")

	args := []any{"dump", "--quiet"}
	if dumpOptions.Snapshot != "" {
		args = append(args, dumpOptions.Snapshot)
	} else {
		args = append(args, "latest")
	}
	if dumpOptions.FileName != "" {
		args = append(args, dumpOptions.FileName)
	} else {
		args = append(args, "stdin")
	}
	if dumpOptions.Host != "" {
		args = append(args, "--host")
		args = append(args, dumpOptions.SourceHost)
	}
	if dumpOptions.Path != "" {
		args = append(args, "--path")
		args = append(args, dumpOptions.Path)
	}

	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)

	// first add restic command, then add StdoutPipeCommands
	commands := []Command{
		{Name: ResticCMD, Args: args},
	}
	commands = append(commands, dumpOptions.StdoutPipeCommands...)
	return w.run(commands...)
}

func (w *ResticWrapper) check() ([]byte, error) {
	klog.Infoln("Checking integrity of repository")
	args := w.appendCacheDirFlag([]any{"check", "--no-lock"})
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)
	args = w.appendMaxConnectionsFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) stats(snapshotID string) ([]byte, error) {
	klog.Infoln("Reading repository status")
	args := w.appendCacheDirFlag([]any{"stats"})
	if snapshotID != "" {
		args = append(args, snapshotID)
	}
	args = w.appendMaxConnectionsFlag(args)
	args = append(args, "--quiet", "--json", "--mode", "raw-data", "--no-lock")
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) unlock() ([]byte, error) {
	klog.Infoln("Unlocking restic repository")
	args := w.appendCacheDirFlag([]any{"unlock", "--remove-all"})
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) migrateToV2() ([]byte, error) {
	klog.Infoln("Migrating repository to v2")
	args := w.appendCacheDirFlag([]any{"migrate", "upgrade_repo_v2"})
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) prune(pruneOpts PruneOptions) ([]byte, error) {
	klog.Infoln("Pruning repository")

	args := []any{"prune"}
	if pruneOpts.DryRun {
		args = append(args, "--dry-run")
	}
	if pruneOpts.RepackCacheableOnly {
		args = append(args, "--repack-cacheable-only")
	}
	if pruneOpts.RepackSmall {
		args = append(args, "--repack-small")
	}
	if pruneOpts.RepackUncompressed {
		args = append(args, "--repack-uncompressed")
	}
	if pruneOpts.MaxUnusedLimit != "" {
		args = append(args,
			fmt.Sprintf("--max-unused=%s", pruneOpts.MaxUnusedLimit))
	}
	if pruneOpts.MaxRepackSize != "" {
		args = append(args,
			fmt.Sprintf("--max-repack-size=%s", pruneOpts.MaxRepackSize))
	}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) appendCacheDirFlag(args []any) []any {
	if w.config.EnableCache {
		cacheDir := filepath.Join(w.config.ScratchDir, resticCacheDir)
		return append(args, "--cache-dir", cacheDir)
	}
	return append(args, "--no-cache")
}

func (w *ResticWrapper) appendMaxConnectionsFlag(args []any) []any {
	var maxConOption string
	if w.config.MaxConnections > 0 {
		switch w.config.Provider {
		case storage.ProviderGCS:
			maxConOption = fmt.Sprintf("gs.connections=%d", w.config.MaxConnections)
		case storage.ProviderAzure:
			maxConOption = fmt.Sprintf("azure.connections=%d", w.config.MaxConnections)
		case storage.ProviderB2:
			maxConOption = fmt.Sprintf("b2.connections=%d", w.config.MaxConnections)
		}
	}
	if maxConOption != "" {
		return append(args, "--option", maxConOption)
	}
	return args
}

func (w *ResticWrapper) appendCleanupCacheFlag(args []any) []any {
	if w.config.EnableCache {
		return append(args, "--cleanup-cache")
	}
	return args
}

func (w *ResticWrapper) appendCaCertFlag(args []any) []any {
	if w.config.CacertFile != "" {
		return append(args, "--cacert", w.config.CacertFile)
	}
	return args
}

func (w *ResticWrapper) run(commands ...Command) ([]byte, error) {
	// write std errors into os.Stderr and buffer
	errBuff, err := circbuf.NewBuffer(256)
	if err != nil {
		return nil, err
	}
	w.sh.Stderr = io.MultiWriter(os.Stderr, errBuff)

	for _, cmd := range commands {
		if cmd.Name == ResticCMD {
			// first apply NiceSettings, then apply IONiceSettings
			cmd, err = w.applyNiceSettings(cmd)
			if err != nil {
				return nil, err
			}
			cmd, err = w.applyIONiceSettings(cmd)
			if err != nil {
				return nil, err
			}
		}

		// Executes Bash commands with "-c" for multiple dump args.
		if cmd.Name == BashCMD {
			w.sh.Command(cmd.Name, "-c", cmd.getCommandArgsAsString())
		} else {
			w.sh.Command(cmd.Name, cmd.Args...)
		}
	}
	out, err := w.sh.Output()
	if err != nil {
		return nil, formatError(err, errBuff.String())
	}
	klog.Infoln("sh-output:", string(out))
	return out, nil
}

// return last line of std error as error reason
func formatError(err error, stdErr string) error {
	parts := strings.Split(strings.TrimSuffix(stdErr, "\n"), "\n")
	if len(parts) > 1 {
		return errors.New(strings.Join(parts[1:], " "))
	}
	return err
}

func (w *ResticWrapper) applyIONiceSettings(oldCommand Command) (Command, error) {
	if w.config.IONice == nil {
		return oldCommand, nil
	}

	// detect "ionice" installation path
	IONiceCMD, err := exec.LookPath("ionice")
	if err != nil {
		return Command{}, err
	}
	newCommand := Command{
		Name: IONiceCMD,
	}
	if w.config.IONice.Class != nil {
		newCommand.Args = append(newCommand.Args, "-c", fmt.Sprint(*w.config.IONice.Class))
	}
	if w.config.IONice.ClassData != nil {
		newCommand.Args = append(newCommand.Args, "-n", fmt.Sprint(*w.config.IONice.ClassData))
	}
	// TODO: should we use "-t" option with ionice ?
	// newCommand.Args = append(newCommand.Args, "-t")

	// append oldCommand as args of newCommand
	newCommand.Args = append(newCommand.Args, oldCommand.Name)
	newCommand.Args = append(newCommand.Args, oldCommand.Args...)
	return newCommand, nil
}

func (w *ResticWrapper) applyNiceSettings(oldCommand Command) (Command, error) {
	if w.config.Nice == nil {
		return oldCommand, nil
	}

	// detect "nice" installation path
	NiceCMD, err := exec.LookPath("nice")
	if err != nil {
		return Command{}, err
	}
	newCommand := Command{
		Name: NiceCMD,
	}
	if w.config.Nice.Adjustment != nil {
		newCommand.Args = append(newCommand.Args, "-n", fmt.Sprint(*w.config.Nice.Adjustment))
	}

	// append oldCommand as args of newCommand
	newCommand.Args = append(newCommand.Args, oldCommand.Name)
	newCommand.Args = append(newCommand.Args, oldCommand.Args...)
	return newCommand, nil
}

func (w *ResticWrapper) addKey(params keyParams) ([]byte, error) {
	klog.Infoln("Adding new key to restic repository")

	args := []any{"key", "add", "--no-lock"}
	if params.host != "" {
		args = append(args, "--host", params.host)
	}

	if params.user != "" {
		args = append(args, "--user", params.user)
	}

	if params.file != "" {
		args = append(args, "--new-password-file", params.file)
	}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) listKey() ([]byte, error) {
	klog.Infoln("Listing restic keys")

	args := []any{"key", "list", "--no-lock"}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) listLocks() ([]byte, error) {
	klog.Infoln("Listing restic locks")

	args := []any{"list", "locks", "--no-lock"}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) lockStats(lockID string) ([]byte, error) {
	klog.Infoln("Getting stats of restic lock")

	args := []any{"cat", "lock", lockID, "--no-lock"}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) updateKey(params keyParams) ([]byte, error) {
	klog.Infoln("Updating restic key")

	args := []any{"key", "passwd", "--no-lock"}

	if params.file != "" {
		args = append(args, "--new-password-file", params.file)
	}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) removeKey(params keyParams) ([]byte, error) {
	klog.Infoln("Removing restic key")

	args := []any{"key", "remove", params.id, "--no-lock"}

	args = w.appendCacheDirFlag(args)
	args = w.appendMaxConnectionsFlag(args)
	args = w.appendCaCertFlag(args)
	args = w.appendInsecureTLSFlag(args)

	return w.run(Command{Name: ResticCMD, Args: args})
}

func (w *ResticWrapper) appendInsecureTLSFlag(args []any) []any {
	if w.config.InsecureTLS {
		return append(args, "--insecure-tls")
	}
	return args
}

func (c *Command) getCommandArgsAsString() string {
	var cmdParts []string
	for _, arg := range c.Args {
		cmdParts = append(cmdParts, fmt.Sprintf("%v", arg))
	}
	return strings.Join(cmdParts, " ")
}

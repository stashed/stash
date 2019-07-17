package restic

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/appscode/go/types"
	"github.com/stretchr/testify/assert"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

var (
	localRepoDir      string
	scratchDir        string
	secretDir         string
	targetDir         string
	password          = "password"
	fileName          = "some-file"
	fileContent       = "hello stash"
	stdinPipeCommand  = Command{Name: "echo", Args: []interface{}{"hello"}}
	stdoutPipeCommand = Command{Name: "cat"}
)

func setupTest(tempDir string) (*ResticWrapper, error) {
	localRepoDir = filepath.Join(tempDir, "repo")
	scratchDir = filepath.Join(tempDir, "scratch")
	secretDir = filepath.Join(tempDir, "secret")
	targetDir = filepath.Join(tempDir, "target")

	if err := os.MkdirAll(localRepoDir, 0777); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(scratchDir, 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(secretDir, 0777); err != nil {
		return nil, err
	}
	err := ioutil.WriteFile(filepath.Join(secretDir, RESTIC_PASSWORD), []byte(password), os.ModePerm)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(targetDir, 0777); err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(filepath.Join(targetDir, fileName), []byte(fileContent), os.ModePerm)
	if err != nil {
		return nil, err
	}

	setupOpt := SetupOptions{
		Provider:    ProviderLocal,
		Path:        localRepoDir,
		SecretDir:   secretDir,
		ScratchDir:  scratchDir,
		EnableCache: false,
	}

	w, err := NewResticWrapper(setupOpt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func cleanup(tempDir string) error {
	if err := os.RemoveAll(tempDir); err != nil {
		return err
	}
	return nil
}

func TestBackupRestoreDirs(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	backupOpt := BackupOptions{
		BackupDirs: []string{targetDir},
		RetentionPolicy: api_v1alpha1.RetentionPolicy{
			Name:     "keep-last-1",
			KeepLast: 1,
			Prune:    true,
			DryRun:   false,
		},
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(backupOut)

	// delete target then restore
	if err = os.RemoveAll(targetDir); err != nil {
		t.Error(err)
	}
	restoreOpt := RestoreOptions{
		RestoreDirs: []string{targetDir},
	}
	restoreOut, err := w.RunRestore(restoreOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(restoreOut)

	// check file
	fileContentByte, err := ioutil.ReadFile(filepath.Join(targetDir, fileName))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, fileContent, string(fileContentByte))
}

func TestBackupRestoreStdin(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	backupOpt := BackupOptions{
		StdinPipeCommand: stdinPipeCommand,
		StdinFileName:    fileName,
		RetentionPolicy: api_v1alpha1.RetentionPolicy{
			Name:     "keep-last-1",
			KeepLast: 1,
			Prune:    true,
			DryRun:   false,
		},
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("backup output:", backupOut)

	dumpOpt := DumpOptions{
		FileName:          fileName,
		StdoutPipeCommand: stdoutPipeCommand,
	}
	dumpOut, err := w.Dump(dumpOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("dump output:", dumpOut)
}

func TestBackupRestoreWithScheduling(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	w.config.IONice = &ofst.IONiceSettings{
		Class:     types.Int32P(2),
		ClassData: types.Int32P(3),
	}
	w.config.Nice = &ofst.NiceSettings{
		Adjustment: types.Int32P(12),
	}

	backupOpt := BackupOptions{
		BackupDirs: []string{targetDir},
		RetentionPolicy: api_v1alpha1.RetentionPolicy{
			Name:     "keep-last-1",
			KeepLast: 1,
			Prune:    true,
			DryRun:   false,
		},
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(backupOut)

	// delete target then restore
	if err = os.RemoveAll(targetDir); err != nil {
		t.Error(err)
	}
	restoreOpt := RestoreOptions{
		RestoreDirs: []string{targetDir},
	}
	restoreOut, err := w.RunRestore(restoreOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(restoreOut)

	// check file
	fileContentByte, err := ioutil.ReadFile(filepath.Join(targetDir, fileName))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, fileContent, string(fileContentByte))
}

func TestBackupRestoreStdinWithScheduling(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	w.config.IONice = &ofst.IONiceSettings{
		Class:     types.Int32P(2),
		ClassData: types.Int32P(3),
	}
	w.config.Nice = &ofst.NiceSettings{
		Adjustment: types.Int32P(12),
	}

	backupOpt := BackupOptions{
		StdinPipeCommand: stdinPipeCommand,
		StdinFileName:    fileName,
		RetentionPolicy: api_v1alpha1.RetentionPolicy{
			Name:     "keep-last-1",
			KeepLast: 1,
			Prune:    true,
			DryRun:   false,
		},
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("backup output:", backupOut)

	dumpOpt := DumpOptions{
		FileName:          fileName,
		StdoutPipeCommand: stdoutPipeCommand,
	}
	dumpOut, err := w.Dump(dumpOpt)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("dump output:", dumpOut)
}

func TestRunParallelBackup(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	// write large (100Mb) sample  file
	largeContent := make([]byte, 104857600)
	fileContent = string(largeContent)

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	backupOpts := newParallelBackupOptions()
	backupOutput, err := w.RunParallelBackup(backupOpts, 2)
	if err != nil {
		t.Error(err)
	}

	// verify repository stats
	assert.Equal(t, *backupOutput.RepositoryStats.Integrity, true)
	assert.Equal(t, backupOutput.RepositoryStats.SnapshotCount, 3)

	// verify each host status
	for i := range backupOutput.HostBackupStats {
		assert.Equal(t, backupOutput.HostBackupStats[i].Phase, api_v1beta1.HostBackupSucceeded)
	}
}

func TestRunParallelRestore(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	// write large (100Mb) sample  file
	largeContent := make([]byte, 104857600)
	fileContent = string(largeContent)

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	backupOpts := newParallelBackupOptions()
	backupOutput, err := w.RunParallelBackup(backupOpts, 2)
	if err != nil {
		t.Error(err)
	}

	// verify that all host backup has succeeded
	for i := range backupOutput.HostBackupStats {
		assert.Equal(t, backupOutput.HostBackupStats[i].Phase, api_v1beta1.HostBackupSucceeded)
	}

	// run parallel restore
	restoreOptions, err := newParallelRestoreOptions(tempDir)
	if err != nil {
		t.Error(err)
	}
	restoreOutput, err := w.RunParallelRestore(restoreOptions, 2)
	if err != nil {
		t.Error(err)
	}

	// verify that all host has been restored successfully
	for i := range restoreOutput.HostRestoreStats {
		assert.Equal(t, restoreOutput.HostRestoreStats[i].Phase, api_v1beta1.HostRestoreSucceeded)
	}

	// verify that restored file contents are identical to the backed up file
	for i := range restoreOptions {
		// check file
		restoredFileContent, err := ioutil.ReadFile(filepath.Join(restoreOptions[i].Destination, targetDir, fileName))
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, fileContent, string(restoredFileContent))
	}
}

func TestRunParallelDump(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "stash-unit-test-")
	if err != nil {
		t.Error(err)
	}

	// write large (100Mb) sample  file
	largeContent := make([]byte, 104857600)
	fileContent = string(largeContent)

	w, err := setupTest(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer cleanup(tempDir)

	backupOpts := newParallelBackupOptions()
	backupOutput, err := w.RunParallelBackup(backupOpts, 2)
	if err != nil {
		t.Error(err)
	}

	// verify that all host backup has succeeded
	for i := range backupOutput.HostBackupStats {
		assert.Equal(t, backupOutput.HostBackupStats[i].Phase, api_v1beta1.HostBackupSucceeded)
	}

	// run parallel dump
	dumpOptions, err := newParallelDumpOptions(tempDir)
	if err != nil {
		t.Error(err)
	}
	dumpOutput, err := w.ParallelDump(dumpOptions, 2)
	if err != nil {
		t.Error(err)
	}

	// verify that all host has been restored successfully
	for i := range dumpOutput.HostRestoreStats {
		assert.Equal(t, dumpOutput.HostRestoreStats[i].Phase, api_v1beta1.HostRestoreSucceeded)
	}
}

func newParallelBackupOptions() []BackupOptions {
	return []BackupOptions{
		{
			Host:       "host-0",
			BackupDirs: []string{targetDir},
			RetentionPolicy: api_v1alpha1.RetentionPolicy{
				Name:     "keep-last-1",
				KeepLast: 1,
				Prune:    true,
				DryRun:   false,
			},
		},
		{
			Host:       "host-1",
			BackupDirs: []string{targetDir},
			RetentionPolicy: api_v1alpha1.RetentionPolicy{
				Name:     "keep-last-1",
				KeepLast: 1,
				Prune:    true,
				DryRun:   false,
			},
		},
		{
			Host:       "host-2",
			BackupDirs: []string{targetDir},
			RetentionPolicy: api_v1alpha1.RetentionPolicy{
				Name:     "keep-last-1",
				KeepLast: 1,
				Prune:    true,
				DryRun:   false,
			},
		},
	}
}

func newParallelRestoreOptions(tempDir string) ([]RestoreOptions, error) {
	if err := os.MkdirAll(filepath.Join(tempDir, "host-0"), 0777); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "host-1"), 0777); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "host-2"), 0777); err != nil {
		return nil, err
	}

	return []RestoreOptions{
		{
			Host:        "host-0",
			SourceHost:  "",
			RestoreDirs: []string{targetDir},
			Destination: filepath.Join(tempDir, "host-0"),
		},
		{
			Host:        "host-1",
			SourceHost:  "",
			RestoreDirs: []string{targetDir},
			Destination: filepath.Join(tempDir, "host-1"),
		},
		{
			Host:        "host-2",
			SourceHost:  "",
			RestoreDirs: []string{targetDir},
			Destination: filepath.Join(tempDir, "host-2"),
		},
	}, nil
}

func newParallelDumpOptions(tempDir string) ([]DumpOptions, error) {

	return []DumpOptions{
		{
			Host:              "host-0",
			FileName:          filepath.Join(targetDir, fileName),
			StdoutPipeCommand: stdoutPipeCommand,
		},
		{
			Host:              "host-1",
			FileName:          filepath.Join(targetDir, fileName),
			StdoutPipeCommand: stdoutPipeCommand,
		},
		{
			Host:              "host-2",
			FileName:          filepath.Join(targetDir, fileName),
			StdoutPipeCommand: stdoutPipeCommand,
		},
	}, nil
}

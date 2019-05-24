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

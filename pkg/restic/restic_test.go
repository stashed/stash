package restic

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	localRepoDir      = "/tmp/stash/repo"
	scratchDir        = "/tmp/stash/scratch"
	secretDir         = "/tmp/stash/secret"
	targetDir         = "/tmp/stash/target"
	password          = "password"
	fileName          = "some-file"
	fileContent       = "hello stash"
	stdinPipeCommand  = Command{Name: "echo", Args: []interface{}{"hello"}}
	stdoutPipeCommand = Command{Name: "cat"}
)

func setupTest() *ResticWrapper {
	if err := os.MkdirAll(localRepoDir, 0777); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(scratchDir, 0777); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(secretDir, 0777); err != nil {
		log.Fatal(err)
	}
	err := ioutil.WriteFile(filepath.Join(secretDir, RESTIC_PASSWORD), []byte(password), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(targetDir, 0777); err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(targetDir, fileName), []byte(fileContent), os.ModePerm)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	return w
}

func cleanup() {
	if err := os.RemoveAll(localRepoDir); err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(scratchDir); err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(secretDir); err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		log.Fatal(err)
	}
}

func TestBackupRestoreDirs(t *testing.T) {
	w := setupTest()
	defer cleanup()

	backupOpt := BackupOptions{
		BackupDirs: []string{targetDir},
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(backupOut)

	// delete target then restore
	if err = os.RemoveAll(targetDir); err != nil {
		log.Fatal(err)
	}
	restoreOpt := RestoreOptions{
		RestoreDirs: []string{targetDir},
	}
	restoreOut, err := w.RunRestore(restoreOpt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(restoreOut)

	// check file
	fileContentByte, err := ioutil.ReadFile(filepath.Join(targetDir, fileName))
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, fileContent, string(fileContentByte))
}

func TestBackupRestoreStdin(t *testing.T) {
	w := setupTest()
	defer cleanup()

	backupOpt := BackupOptions{
		StdinPipeCommand: stdinPipeCommand,
		StdinFileName:    fileName,
	}
	backupOut, err := w.RunBackup(backupOpt)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("backup output:", backupOut)

	dumpOpt := DumpOptions{
		FileName:          fileName,
		StdoutPipeCommand: stdoutPipeCommand,
	}
	dumpOut, err := w.Dump(dumpOpt)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("dump output:", dumpOut)
}

package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appscode/go/types"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

type BackupOutput struct {
	// HostBackupStats shows backup statistics of current host
	HostBackupStats api_v1beta1.HostBackupStats `json:"hostBackupStats,omitempty"`
	// RepositoryStats shows statistics of repository after last backup
	RepositoryStats RepositoryStats `json:"repository,omitempty"`
}

type RepositoryStats struct {
	// Integrity shows result of repository integrity check after last backup
	Integrity *bool `json:"integrity,omitempty"`
	// Size show size of repository after last backup
	Size string `json:"size,omitempty"`
	// SnapshotCount shows number of snapshots stored in the repository
	SnapshotCount int `json:"snapshotCount,omitempty"`
	// SnapshotsRemovedOnLastCleanup shows number of old snapshots cleaned up according to retention policy on last backup session
	SnapshotsRemovedOnLastCleanup int `json:"snapshotsRemovedOnLastCleanup,omitempty"`
}

type RestoreOutput struct {
	// HostRestoreStats shows restore statistics of current host
	HostRestoreStats api_v1beta1.HostRestoreStats `json:"hostRestoreStats,omitempty"`
}

// WriteOutput write output of backup process into output.json file in the directory
// specified by outputDir parameter
func (out *BackupOutput) WriteOutput(fileName string) error {
	jsonOutput, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func (out *RestoreOutput) WriteOutput(fileName string) error {
	jsonOutput, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func ReadBackupOutput(filename string) (*BackupOutput, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	backupOutput := &BackupOutput{}
	err = json.Unmarshal(data, backupOutput)
	if err != nil {
		return nil, err
	}

	return backupOutput, nil
}

func ReadRestoreOutput(filename string) (*RestoreOutput, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	restoreOutput := &RestoreOutput{}
	err = json.Unmarshal(data, restoreOutput)
	if err != nil {
		return nil, err
	}

	return restoreOutput, nil
}

// ExtractBackupInfo extract information from output of "restic backup" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) extractBackupInfo(output []byte, directory string, host string) error {
	// unmarshal json output
	var jsonOutput BackupSummary
	dec := json.NewDecoder(bytes.NewReader(output))
	for {

		err := dec.Decode(&jsonOutput)
		if err == io.EOF {
			// all done
			break
		}
		if err != nil {
			return err
		}
		// if message type is summary then we have found our desired message block
		if jsonOutput.MessageType == "summary" {
			break
		}
	}

	snapshotStats := api_v1beta1.SnapshotStats{
		Directory: directory,
	}

	snapshotStats.FileStats.NewFiles = jsonOutput.FilesNew
	snapshotStats.FileStats.ModifiedFiles = jsonOutput.FilesChanged
	snapshotStats.FileStats.UnmodifiedFiles = jsonOutput.FilesUnmodified
	snapshotStats.FileStats.TotalFiles = jsonOutput.TotalFilesProcessed

	snapshotStats.Uploaded = formatBytes(jsonOutput.DataAdded)
	snapshotStats.Size = formatBytes(jsonOutput.TotalBytesProcessed)
	snapshotStats.ProcessingTime = formatSeconds(uint64(jsonOutput.TotalDuration))
	snapshotStats.Name = jsonOutput.SnapshotID

	// if there is already an entry for this directory then update that
	for i, v := range backupOutput.HostBackupStats.Snapshots {
		if v.Directory == snapshotStats.Directory {
			backupOutput.HostBackupStats.Snapshots[i] = snapshotStats
			return nil
		}
	}

	// new entry. so append to backupOutput.
	backupOutput.HostBackupStats.Snapshots = append(backupOutput.HostBackupStats.Snapshots, snapshotStats)

	return nil
}

// ExtractCheckInfo extract information from output of "restic check" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) extractCheckInfo(out []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		if line == "no errors were found" {
			backupOutput.RepositoryStats.Integrity = types.BoolP(true)
			return
		}
	}
	backupOutput.RepositoryStats.Integrity = types.BoolP(false)
}

// ExtractCleanupInfo extract information from output of "restic forget" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) extractCleanupInfo(out []byte) error {
	var fg []ForgetGroup
	err := json.Unmarshal(out, &fg)
	if err != nil {
		return err
	}

	backupOutput.RepositoryStats.SnapshotsRemovedOnLastCleanup = 0
	backupOutput.RepositoryStats.SnapshotCount = 0
	for i := 0; i < len(fg); i++ {
		backupOutput.RepositoryStats.SnapshotsRemovedOnLastCleanup += len(fg[i].Remove)
		backupOutput.RepositoryStats.SnapshotCount += len(fg[i].Keep)
	}

	return nil
}

// ExtractStatsInfo extract information from output of "restic stats" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) extractStatsInfo(out []byte) error {
	var stat StatsContainer
	err := json.Unmarshal(out, &stat)
	if err != nil {
		return err
	}
	backupOutput.RepositoryStats.Size = formatBytes(stat.TotalSize)
	return nil
}

type BackupSummary struct {
	MessageType         string  `json:"message_type"` // "summary"
	FilesNew            *int    `json:"files_new"`
	FilesChanged        *int    `json:"files_changed"`
	FilesUnmodified     *int    `json:"files_unmodified"`
	DataAdded           uint64  `json:"data_added"`
	TotalFilesProcessed *int    `json:"total_files_processed"`
	TotalBytesProcessed uint64  `json:"total_bytes_processed"`
	TotalDuration       float64 `json:"total_duration"` // in seconds
	SnapshotID          string  `json:"snapshot_id"`
}

type ForgetGroup struct {
	Keep   []json.RawMessage `json:"keep"`
	Remove []json.RawMessage `json:"remove"`
}

type StatsContainer struct {
	TotalSize uint64 `json:"total_size"`
}

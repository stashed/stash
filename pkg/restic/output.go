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

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

type BackupOutput struct {
	// HostBackupStats shows backup statistics of a host
	HostBackupStats []api_v1beta1.HostBackupStats `json:"hostBackupStats,omitempty"`
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
	// HostRestoreStats shows restore statistics of a host
	HostRestoreStats []api_v1beta1.HostRestoreStats `json:"hostRestoreStats,omitempty"`
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
func extractBackupInfo(output []byte, path string) (api_v1beta1.SnapshotStats, error) {
	snapshotStats := api_v1beta1.SnapshotStats{
		Path: path,
	}

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
			return snapshotStats, err
		}
		// if message type is summary then we have found our desired message block
		if jsonOutput.MessageType == "summary" {
			break
		}
	}

	snapshotStats.FileStats.NewFiles = jsonOutput.FilesNew
	snapshotStats.FileStats.ModifiedFiles = jsonOutput.FilesChanged
	snapshotStats.FileStats.UnmodifiedFiles = jsonOutput.FilesUnmodified
	snapshotStats.FileStats.TotalFiles = jsonOutput.TotalFilesProcessed

	snapshotStats.Uploaded = formatBytes(jsonOutput.DataAdded)
	snapshotStats.Size = formatBytes(jsonOutput.TotalBytesProcessed)
	snapshotStats.ProcessingTime = formatSeconds(uint64(jsonOutput.TotalDuration))
	snapshotStats.Name = jsonOutput.SnapshotID

	return snapshotStats, nil
}

// ExtractCheckInfo extract information from output of "restic check" command and
// save valuable information into backupOutput
func extractCheckInfo(out []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		if line == "no errors were found" {
			return true
		}
	}
	return false
}

// ExtractCleanupInfo extract information from output of "restic forget" command and
// save valuable information into backupOutput
func extractCleanupInfo(out []byte) (int, int, error) {
	var fg []ForgetGroup
	err := json.Unmarshal(out, &fg)
	if err != nil {
		return 0, 0, err
	}

	keep := 0
	removed := 0
	for i := 0; i < len(fg); i++ {
		keep += len(fg[i].Keep)
		removed += len(fg[i].Remove)
	}

	return keep, removed, nil
}

// ExtractStatsInfo extract information from output of "restic stats" command and
// save valuable information into backupOutput
func extractStatsInfo(out []byte) (string, error) {
	var stat StatsContainer
	err := json.Unmarshal(out, &stat)
	if err != nil {
		return "", err
	}
	return formatBytes(stat.TotalSize), nil
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

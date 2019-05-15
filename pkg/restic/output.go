package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	var line string
	scanner := bufio.NewScanner(bytes.NewReader(output))

	snapshotStats := api_v1beta1.SnapshotStats{
		Directory: directory,
	}

	for scanner.Scan() {
		line = scanner.Text()
		if strings.HasPrefix(line, "Files:") {
			info := strings.FieldsFunc(line, separators)
			if len(info) < 7 {
				return fmt.Errorf("failed to parse files statistics")
			}
			newFiles, err := strconv.Atoi(info[1])
			if err != nil {
				return err
			}
			modifiedFiles, err := strconv.Atoi(info[3])
			if err != nil {
				return err
			}
			unmodifiedFiles, err := strconv.Atoi(info[5])
			if err != nil {
				return err
			}
			snapshotStats.FileStats.NewFiles = types.IntP(newFiles)
			snapshotStats.FileStats.ModifiedFiles = types.IntP(modifiedFiles)
			snapshotStats.FileStats.UnmodifiedFiles = types.IntP(unmodifiedFiles)
		} else if strings.HasPrefix(line, "Added to the repo:") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 6 {
				return fmt.Errorf("failed to parse upload statistics")
			}
			snapshotStats.Uploaded = info[length-2] + " " + info[length-1]
		} else if strings.HasPrefix(line, "processed") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 7 {
				return fmt.Errorf("failed to parse file processing statistics")
			}
			totalFiles, err := strconv.Atoi(info[1])
			if err != nil {
				return err
			}
			snapshotStats.FileStats.TotalFiles = types.IntP(totalFiles)
			snapshotStats.Size = info[3] + " " + info[4]
			m, s, err := convertToMinutesSeconds(info[6])
			if err != nil {
				return err
			}
			snapshotStats.ProcessingTime = fmt.Sprintf("%dm%ds", m, s)
		} else if strings.HasPrefix(line, "snapshot") && strings.HasSuffix(line, "saved") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 3 {
				return fmt.Errorf("failed to parse snapshot statistics")
			}
			snapshotStats.Name = info[1]
		}
	}

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
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var line string
	snapshotCount := 0
	snapshotRemoved := 0
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "keep") && strings.HasSuffix(line, "snapshots:") {
			parts := strings.FieldsFunc(line, separators)
			length := len(parts)
			if length < 3 {
				return fmt.Errorf("failed to parse current available snapshot statistics")
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				return err
			}
			snapshotCount = snapshotCount + c
		}
		if strings.HasPrefix(line, "remove") && strings.HasSuffix(line, "snapshots:") {
			parts := strings.FieldsFunc(line, separators)
			length := len(parts)
			if length < 3 {
				return fmt.Errorf("failed to parse cleaned snapshot statistics")
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				return err
			}
			snapshotRemoved = snapshotRemoved + c
		}
	}
	backupOutput.RepositoryStats.SnapshotsRemovedOnLastCleanup = snapshotRemoved
	backupOutput.RepositoryStats.SnapshotCount = snapshotCount
	return nil
}

// ExtractStatsInfo extract information from output of "restic stats" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) extractStatsInfo(out []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Total Size:") {
			parts := strings.FieldsFunc(line, separators)
			length := len(parts)
			if length < 4 {
				return fmt.Errorf("failed to parse repository statistics")
			}

			backupOutput.RepositoryStats.Size = parts[2] + " " + parts[3]
		}
	}
	return nil
}

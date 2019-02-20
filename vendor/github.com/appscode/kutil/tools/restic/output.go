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
)

type BackupOutput struct {
	// BackupStats shows statistics of last backup session
	BackupStats BackupStats `json:"backup,omitempty"`
	// RepositoryStats shows statistics of repository after last backup
	RepositoryStats RepositoryStats `json:"repository,omitempty"`
}

type BackupStats struct {
	// Snapshot indicates the name of the backup snapshot created in this backup session
	Snapshot string `json:"snapshot,omitempty"`
	// Size indicates the size of target data to backup
	Size string `json:"size,omitempty"`
	// Uploaded indicates size of data uploaded to backend in this backup session
	Uploaded string `json:"uploaded,omitempty"`
	// ProcessingTime indicates time taken to process the target data
	ProcessingTime string `json:"processingTime,omitempty"`
	// FileStats shows statistics of files of backup session
	FileStats FileStats `json:"fileStats,omitempty"`
}
type RepositoryStats struct {
	// Integrity shows result of repository integrity check after last backup
	Integrity *bool `json:"integrity,omitempty"`
	// Size show size of repository after last backup
	Size string `json:"size,omitempty"`
	// SnapshotCount shows number of snapshots stored in the repository
	SnapshotCount int `json:"snapshotCount,omitempty"`
	// SnapshotRemovedOnLastCleanup shows number of old snapshots cleaned up according to retention policy on last backup session
	SnapshotRemovedOnLastCleanup int `json:"snapshotRemovedOnLastCleanup,omitempty"`
}

type FileStats struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles *int `json:"totalFiles,omitempty"`
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles *int `json:"newFiles,omitempty"`
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles *int `json:"modifiedFiles,omitempty"`
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles *int `json:"unmodifiedFiles,omitempty"`
}

// WriteOutput write output of backup process into output.json file in the directory
// specified by outputDir parameter
func WriteOutput(out *BackupOutput, outputDir string) error {
	jsonOuput, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	fileName := filepath.Join(outputDir, "output.json")
	if err := ioutil.WriteFile(fileName, jsonOuput, 0755); err != nil {
		return err
	}
	return nil
}

// ExtractBackupInfo extract information from output of "restic backup" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) ExtractBackupInfo(output []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var line string
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
			backupOutput.BackupStats.FileStats.NewFiles = types.IntP(newFiles)
			backupOutput.BackupStats.FileStats.ModifiedFiles = types.IntP(modifiedFiles)
			backupOutput.BackupStats.FileStats.UnmodifiedFiles = types.IntP(unmodifiedFiles)
		} else if strings.HasPrefix(line, "Added to the repo:") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 6 {
				return fmt.Errorf("failed to parse upload statistics")
			}
			backupOutput.BackupStats.Uploaded = info[length-2] + " " + info[length-1]
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
			backupOutput.BackupStats.FileStats.TotalFiles = types.IntP(totalFiles)
			backupOutput.BackupStats.Size = info[3] + " " + info[4]
			m, s, err := convertToMinutesSeconds(info[6])
			if err != nil {
				return err
			}
			backupOutput.BackupStats.ProcessingTime = fmt.Sprintf("%dm%ds", m, s)
		} else if strings.HasPrefix(line, "snapshot") && strings.HasSuffix(line, "saved") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 3 {
				return fmt.Errorf("failed to parse snapshot statistics")
			}
			backupOutput.BackupStats.Snapshot = info[1]
		}
	}
	return nil
}

// ExtractCheckInfo extract information from output of "restic check" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) ExtractCheckInfo(out []byte) {
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
func (backupOutput *BackupOutput) ExtractCleanupInfo(out []byte) error {
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
	backupOutput.RepositoryStats.SnapshotRemovedOnLastCleanup = snapshotRemoved
	backupOutput.RepositoryStats.SnapshotCount = snapshotCount
	return nil
}

// ExtractStatsInfo extract information from output of "restic stats" command and
// save valuable information into backupOutput
func (backupOutput *BackupOutput) ExtractStatsInfo(out []byte) error {
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

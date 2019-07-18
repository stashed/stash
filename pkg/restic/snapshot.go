package restic

import "encoding/json"

func (w *ResticWrapper) ListSnapshots(snapshotIDs []string) ([]Snapshot, error) {
	return w.listSnapshots(snapshotIDs)
}

func (w *ResticWrapper) DeleteSnapshots(snapshotIDs []string) ([]byte, error) {
	return w.deleteSnapshots(snapshotIDs)
}

// GetSnapshotSize returns size of a snapshot in bytes
func (w *ResticWrapper) GetSnapshotSize(snapshotID string) (uint64, error) {
	out, err := w.stats(snapshotID)
	if err != nil {
		return 0, err
	}

	var stat StatsContainer
	err = json.Unmarshal(out, &stat)
	if err != nil {
		return 0, err
	}
	return stat.TotalSize, nil

}

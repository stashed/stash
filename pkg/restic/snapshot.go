package restic

func (w *ResticWrapper) ListSnapshots(snapshotIDs []string) ([]Snapshot, error) {
	return w.listSnapshots(snapshotIDs)
}

func (w *ResticWrapper) DeleteSnapshots(snapshotIDs []string) ([]byte, error) {
	return w.deleteSnapshots(snapshotIDs)
}

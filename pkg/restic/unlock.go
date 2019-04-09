package restic

func (w *ResticWrapper) UnlockRepository() error {
	_, err := w.unlock()
	return err
}

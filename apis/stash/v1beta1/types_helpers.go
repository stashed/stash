package v1beta1

const (
	StashBackupComponent = "backup"
	StashRestoreComponent= "restore"
)

// TODO: complete
func (t TargetRef) IsWorkload() bool {
	if t.Kind == "Deployment" {
		return true
	}
	return false
}

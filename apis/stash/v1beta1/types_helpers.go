package v1beta1

// TODO: complete
func (t TargetRef) IsWorkload() bool {
	if t.Kind == "Deployment" {
		return true
	}
	return false
}

/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

const (
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	StashKey = "stash.appscode.com"

	KeyBackupBlueprint = StashKey + "/backup-blueprint"
	KeyTargetPaths     = StashKey + "/target-paths"
	KeyVolumeMounts    = StashKey + "/volume-mounts"

	KeyLastAppliedRestoreSession    = StashKey + "/last-applied-restoresession"
	KeyLastAppliedBackupInvoker     = StashKey + "/last-applied-backup-invoker"
	KeyLastAppliedBackupInvokerKind = StashKey + "/last-applied-backup-invoker-kind"

	AppliedBackupInvokerSpecHash  = StashKey + "/last-applied-backup-invoker-hash"
	AppliedRestoreSessionSpecHash = StashKey + "/last-applied-restoresession-hash"
)

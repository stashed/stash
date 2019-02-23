/*
Copyright 2019 The Stash Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1beta1

import (
	internalinterfaces "github.com/appscode/stash/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// BackupConfigurations returns a BackupConfigurationInformer.
	BackupConfigurations() BackupConfigurationInformer
	// BackupSessions returns a BackupSessionInformer.
	BackupSessions() BackupSessionInformer
	// BackupTemplates returns a BackupTemplateInformer.
	BackupTemplates() BackupTemplateInformer
	// Functions returns a FunctionInformer.
	Functions() FunctionInformer
	// RestoreSessions returns a RestoreSessionInformer.
	RestoreSessions() RestoreSessionInformer
	// Tasks returns a TaskInformer.
	Tasks() TaskInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// BackupConfigurations returns a BackupConfigurationInformer.
func (v *version) BackupConfigurations() BackupConfigurationInformer {
	return &backupConfigurationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// BackupSessions returns a BackupSessionInformer.
func (v *version) BackupSessions() BackupSessionInformer {
	return &backupSessionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// BackupTemplates returns a BackupTemplateInformer.
func (v *version) BackupTemplates() BackupTemplateInformer {
	return &backupTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Functions returns a FunctionInformer.
func (v *version) Functions() FunctionInformer {
	return &functionInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// RestoreSessions returns a RestoreSessionInformer.
func (v *version) RestoreSessions() RestoreSessionInformer {
	return &restoreSessionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Tasks returns a TaskInformer.
func (v *version) Tasks() TaskInformer {
	return &taskInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

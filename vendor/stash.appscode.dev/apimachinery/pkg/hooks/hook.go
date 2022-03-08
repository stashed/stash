/*
Copyright AppsCode Inc. and Contributors

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

package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	prober "kmodules.xyz/prober/api/v1"
	"kmodules.xyz/prober/probe"
)

type HookExecutor struct {
	Config      *rest.Config
	Hook        *prober.Handler
	ExecutorPod kmapi.ObjectReference
	Summary     *v1beta1.Summary
}

func (e *HookExecutor) Execute() error {
	if strings.Contains(e.Hook.String(), "{{") {
		if err := e.renderTemplate(); err != nil {
			return err
		}
	}
	return probe.RunProbe(e.Config, e.Hook, e.ExecutorPod.Name, e.ExecutorPod.Namespace)
}

var pool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (e *HookExecutor) renderTemplate() error {
	hookContent, err := json.Marshal(e.Hook)
	if err != nil {
		return err
	}

	tpl, err := template.New("hook-template").Parse(string(hookContent))
	if err != nil {
		return err
	}
	tpl.Option("missingkey=default")

	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pool.Put(buf)

	err = tpl.Execute(buf, e.Summary)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), &e.Hook)
}

type BackupHookExecutor struct {
	Config        *rest.Config
	StashClient   cs.Interface
	BackupSession *v1beta1.BackupSession
	Invoker       invoker.BackupInvoker
	Target        v1beta1.TargetRef
	ExecutorPod   kmapi.ObjectReference
	Hook          *prober.Handler
	HookType      string
}

func (e *BackupHookExecutor) Execute() error {
	hookExecutor := HookExecutor{
		Config:      e.Config,
		Hook:        e.Hook,
		ExecutorPod: e.ExecutorPod,
		Summary: e.Invoker.GetSummary(e.Target, kmapi.ObjectReference{
			Namespace: e.BackupSession.Namespace,
			Name:      e.BackupSession.Name,
		}),
	}

	session := invoker.NewBackupSessionHandler(e.StashClient, e.BackupSession)
	if e.alreadyExecuted(session) {
		return e.skipHookReExecution(session)
	}

	if err := hookExecutor.Execute(); err != nil {
		condErr := e.setBackupHookExecutionSucceededToFalse(session, err)
		return errors.NewAggregate([]error{err, condErr})
	}
	return e.setBackupHookExecutionSucceededToTrue(session)
}

func (e *BackupHookExecutor) alreadyExecuted(session *invoker.BackupSessionHandler) bool {
	if e.HookType == apis.PreBackupHook {
		return kmapi.HasCondition(session.GetTargetConditions(e.Target), v1beta1.PreBackupHookExecutionSucceeded)
	}
	return kmapi.HasCondition(session.GetTargetConditions(e.Target), v1beta1.PostBackupHookExecutionSucceeded)
}

func (e *BackupHookExecutor) skipHookReExecution(session *invoker.BackupSessionHandler) error {
	klog.Infof("Skipping executing %s. Reason: It has been executed already....", e.HookType)

	var cond *kmapi.Condition
	targetConditions := session.GetTargetConditions(e.Target)

	if e.HookType == apis.PreBackupHook {
		_, cond = kmapi.GetCondition(targetConditions, v1beta1.PreBackupHookExecutionSucceeded)
	} else {
		_, cond = kmapi.GetCondition(targetConditions, v1beta1.PostBackupHookExecutionSucceeded)
	}
	if cond != nil && cond.Status == corev1.ConditionFalse {
		return fmt.Errorf("%s hook failed to execute. Reason: ", cond.Reason)
	}
	return nil
}

func (e *BackupHookExecutor) setBackupHookExecutionSucceededToFalse(session *invoker.BackupSessionHandler, err error) error {
	if e.HookType == apis.PreBackupHook {
		return conditions.SetPreBackupHookExecutionSucceededToFalse(session, e.Target, err)
	}
	return conditions.SetPostBackupHookExecutionSucceededToFalse(session, e.Target, err)
}

func (e *BackupHookExecutor) setBackupHookExecutionSucceededToTrue(session *invoker.BackupSessionHandler) error {
	if e.HookType == apis.PreBackupHook {
		return conditions.SetPreBackupHookExecutionSucceededToTrue(session, e.Target)
	}
	return conditions.SetPostBackupHookExecutionSucceededToTrue(session, e.Target)
}

type RestoreHookExecutor struct {
	Config      *rest.Config
	Invoker     invoker.RestoreInvoker
	Target      v1beta1.TargetRef
	ExecutorPod kmapi.ObjectReference
	Hook        *prober.Handler
	HookType    string
}

func (e *RestoreHookExecutor) Execute() error {
	hookExecutor := HookExecutor{
		Config:      e.Config,
		Hook:        e.Hook,
		ExecutorPod: e.ExecutorPod,
		Summary: e.Invoker.GetSummary(e.Target, kmapi.ObjectReference{
			Namespace: e.Invoker.GetObjectMeta().Namespace,
			Name:      e.Invoker.GetObjectMeta().Name,
		}),
	}

	previouslyExecuted, err := e.alreadyExecuted()
	if err != nil {
		return err
	}
	if previouslyExecuted {
		return e.skipHookReExecution()
	}

	if err := hookExecutor.Execute(); err != nil {
		condErr := e.setRestoreHookExecutionSucceededToFalse(err)
		return errors.NewAggregate([]error{err, condErr})
	}
	return e.setRestoreHookExecutionSucceededToTrue()
}

func (e *RestoreHookExecutor) alreadyExecuted() (bool, error) {
	if e.HookType == apis.PreRestoreHook {
		return e.Invoker.HasCondition(&e.Target, v1beta1.PreRestoreHookExecutionSucceeded)
	}
	return e.Invoker.HasCondition(&e.Target, v1beta1.PostRestoreHookExecutionSucceeded)
}

func (e *RestoreHookExecutor) skipHookReExecution() error {
	klog.Infof("Skipping executing %s. Reason: It has been executed already....", e.HookType)

	var cond *kmapi.Condition
	if e.HookType == apis.PreRestoreHook {
		_, c, err := e.Invoker.GetCondition(&e.Target, v1beta1.PreRestoreHookExecutionSucceeded)
		if err != nil {
			return err
		}
		cond = c
	} else {
		_, c, err := e.Invoker.GetCondition(&e.Target, v1beta1.PostRestoreHookExecutionSucceeded)
		if err != nil {
			return err
		}
		cond = c
	}
	if cond != nil && cond.Status == corev1.ConditionFalse {
		return fmt.Errorf("%s hook failed to execute. Reason: ", cond.Reason)
	}
	return nil
}

func (e *RestoreHookExecutor) setRestoreHookExecutionSucceededToFalse(err error) error {
	if e.HookType == apis.PreRestoreHook {
		return conditions.SetPreRestoreHookExecutionSucceededToFalse(e.Invoker, err)
	}
	return conditions.SetPostRestoreHookExecutionSucceededToFalse(e.Invoker, err)
}

func (e *RestoreHookExecutor) setRestoreHookExecutionSucceededToTrue() error {
	if e.HookType == apis.PreRestoreHook {
		return conditions.SetPreRestoreHookExecutionSucceededToTrue(e.Invoker)
	}
	return conditions.SetPostRestoreHookExecutionSucceededToTrue(e.Invoker)
}

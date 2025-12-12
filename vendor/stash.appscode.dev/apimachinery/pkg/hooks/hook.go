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
	"fmt"
	"strings"
	"sync"
	"text/template"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	sprig "github.com/Masterminds/sprig/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	cutil "kmodules.xyz/client-go/conditions"
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
	if err := e.renderHookTemplate(); err != nil {
		return err
	}
	return probe.RunProbe(e.Config, e.Hook, e.ExecutorPod.Name, e.ExecutorPod.Namespace)
}

func (e *HookExecutor) renderHookTemplate() error {
	if e.Hook.Exec != nil {
		if err := e.renderExecCommand(); err != nil {
			return err
		}
	}

	if e.Hook.HTTPPost != nil {
		if err := e.renderHTTPPostBody(); err != nil {
			return err
		}
	}
	return nil
}

func (e *HookExecutor) renderExecCommand() error {
	for idx, cmd := range e.Hook.Exec.Command {
		if strings.Contains(cmd, "{{") {
			rendered, err := e.renderTemplate(cmd)
			if err != nil {
				return err
			}
			e.Hook.Exec.Command[idx] = rendered
		}
	}
	return nil
}

func (e *HookExecutor) renderHTTPPostBody() error {
	if strings.Contains(e.Hook.HTTPPost.Body, "{{") {
		rendered, err := e.renderTemplate(e.Hook.HTTPPost.Body)
		if err != nil {
			return err
		}
		e.Hook.HTTPPost.Body = rendered
	}
	return nil
}

var pool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func (e *HookExecutor) renderTemplate(text string) (string, error) {
	tpl, err := template.New("hook-template").
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=default").
		Parse(text)
	if err != nil {
		return "", err
	}

	buf := pool.Get().(*bytes.Buffer)
	defer pool.Put(buf)
	buf.Reset()

	if err := tpl.Execute(buf, e.Summary); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

type BackupHookExecutor struct {
	Config          *rest.Config
	StashClient     cs.Interface
	BackupSession   *v1beta1.BackupSession
	Invoker         invoker.BackupInvoker
	Target          v1beta1.TargetRef
	ExecutorPod     kmapi.ObjectReference
	Hook            *prober.Handler
	HookType        string
	ExecutionPolicy v1beta1.HookExecutionPolicy
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

	if !IsAllowedByExecutionPolicy(e.ExecutionPolicy, hookExecutor.Summary) {
		reason := fmt.Sprintf("Skipping executing %s. Reason: executionPolicy is %q but phase is %q.",
			e.HookType,
			getExecutionPolicyWithDefault(e.ExecutionPolicy),
			getTargetPhase(hookExecutor.Summary),
		)
		return e.skipHookExecution(session, reason)
	}

	if e.alreadyExecuted(session) {
		return e.skipHookReExecution(session)
	}

	if err := hookExecutor.Execute(); err != nil {
		condErr := e.setBackupHookExecutionSucceededToFalse(session, err)
		return errors.NewAggregate([]error{err, condErr})
	}
	return e.setBackupHookExecutionSucceededToTrue(session)
}

func IsAllowedByExecutionPolicy(executionPolicy v1beta1.HookExecutionPolicy, summary *v1beta1.Summary) bool {
	if summary == nil {
		return false
	}
	if executionPolicy == v1beta1.ExecuteOnFailure && getTargetPhase(summary) != string(v1beta1.TargetBackupFailed) {
		return false
	}
	if executionPolicy == v1beta1.ExecuteOnSuccess && getTargetPhase(summary) != string(v1beta1.TargetBackupSucceeded) {
		return false
	}
	if executionPolicy == v1beta1.ExecuteOnRetryFailure && (getTargetPhase(summary) != string(v1beta1.TargetBackupFailed) || summary.RetryLeft != 0) {
		return false
	}
	return true
}

func getExecutionPolicyWithDefault(executionPolicy v1beta1.HookExecutionPolicy) v1beta1.HookExecutionPolicy {
	if executionPolicy == "" {
		return v1beta1.ExecuteAlways
	}
	return executionPolicy
}

func getTargetPhase(summary *v1beta1.Summary) string {
	if summary == nil {
		return ""
	}
	return summary.Status.Phase
}

func (e *BackupHookExecutor) alreadyExecuted(session *invoker.BackupSessionHandler) bool {
	if e.HookType == apis.PreBackupHook {
		return cutil.HasCondition(session.GetTargetConditions(e.Target), v1beta1.PreBackupHookExecutionSucceeded)
	}
	return cutil.HasCondition(session.GetTargetConditions(e.Target), v1beta1.PostBackupHookExecutionSucceeded)
}

func (e *BackupHookExecutor) skipHookExecution(session *invoker.BackupSessionHandler, reason string) error {
	klog.Infoln(reason)
	return conditions.SetPostBackupHookExecutionSucceededToTrueWithMsg(session, e.Target, reason)
}

func (e *BackupHookExecutor) skipHookReExecution(session *invoker.BackupSessionHandler) error {
	klog.Infof("Skipping executing %s. Reason: It has been executed already....", e.HookType)

	var cond *kmapi.Condition
	targetConditions := session.GetTargetConditions(e.Target)

	if e.HookType == apis.PreBackupHook {
		_, cond = cutil.GetCondition(targetConditions, v1beta1.PreBackupHookExecutionSucceeded)
	} else {
		_, cond = cutil.GetCondition(targetConditions, v1beta1.PostBackupHookExecutionSucceeded)
	}
	if cond != nil && cond.Status == metav1.ConditionFalse {
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
	Config          *rest.Config
	Invoker         invoker.RestoreInvoker
	Target          v1beta1.TargetRef
	ExecutorPod     kmapi.ObjectReference
	Hook            *prober.Handler
	HookType        string
	ExecutionPolicy v1beta1.HookExecutionPolicy
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

	if !IsAllowedByExecutionPolicy(e.ExecutionPolicy, hookExecutor.Summary) {
		reason := fmt.Sprintf("Skipping executing %s. Reason: executionPolicy is %q but phase is %q.",
			e.HookType,
			getExecutionPolicyWithDefault(e.ExecutionPolicy),
			getTargetPhase(hookExecutor.Summary),
		)
		return e.skipHookExecution(reason)
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

func (e *RestoreHookExecutor) skipHookExecution(msg string) error {
	klog.Infoln(msg)
	return conditions.SetPostRestoreHookExecutionSucceededToTrueWithMsg(e.Invoker, msg)
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
	if cond != nil && cond.Status == metav1.ConditionFalse {
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

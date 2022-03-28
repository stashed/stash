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

package admissionregistration

import (
	"errors"

	"github.com/spf13/pflag"
)

const (
	KeyAdmissionWebhookActive = "admission-webhook.appscode.com/active"
	KeyAdmissionWebhookStatus = "admission-webhook.appscode.com/status"
)

var bypassValidatingWebhookXray = false

func init() {
	pflag.BoolVar(&bypassValidatingWebhookXray, "bypass-validating-webhook-xray", bypassValidatingWebhookXray, "if true, bypasses validating webhook xray checks")
}

var (
	ErrMissingKind         = errors.New("test object missing kind")
	ErrMissingVersion      = errors.New("test object missing version")
	ErrWebhookNotActivated = errors.New("Admission webhooks are not activated. Enable it by configuring --enable-admission-plugins flag of kube-apiserver. For details, visit: https://appsco.de/kube-apiserver-webhooks")
)

func BypassValidatingWebhookXray() bool {
	return bypassValidatingWebhookXray
}

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

package kutil

import (
	"errors"
	"regexp"
	"time"

	kerr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	RetryInterval    = 50 * time.Millisecond
	RetryTimeout     = 2 * time.Second
	ReadinessTimeout = 10 * time.Minute
	GCTimeout        = 5 * time.Minute
)

type VerbType string

const (
	VerbUnchanged VerbType = ""
	VerbCreated   VerbType = "created"
	VerbPatched   VerbType = "patched"
	VerbUpdated   VerbType = "updated"
	VerbDeleted   VerbType = "deleted"

	ObjectNameField = "metadata.name"
)

var (
	ErrNotFound = errors.New("not found")
	ErrUnknown  = errors.New("unknown")
)

func IsRequestRetryable(err error) bool {
	return kerr.IsServiceUnavailable(err) ||
		kerr.IsTimeout(err) ||
		kerr.IsServerTimeout(err) ||
		kerr.IsTooManyRequests(err)
}

var reMutator = regexp.MustCompile(`^Internal error occurred: admission webhook "[^"]+" denied the request.*$`)
var reValidator = regexp.MustCompile(`^admission webhook "[^"]+" denied the request.*$`)

func AdmissionWebhookDeniedRequest(err error) bool {
	return (kerr.IsInternalError(err) && reMutator.MatchString(err.Error())) ||
		(kerr.IsForbidden(err) && reValidator.MatchString(err.Error()))
}

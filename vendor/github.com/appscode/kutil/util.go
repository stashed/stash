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

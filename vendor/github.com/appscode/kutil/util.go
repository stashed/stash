package kutil

import (
	"errors"
	"time"
)

const (
	RetryInterval    = 50 * time.Millisecond
	RetryTimeout     = 2 * time.Second
	ReadinessTimeout = 10 * time.Minute
)

type VerbType string

const (
	VerbUnchanged VerbType = ""
	VerbCreated   VerbType = "created"
	VerbPatched   VerbType = "patched"
	VerbUpdated   VerbType = "updated"
	VerbDeleted   VerbType = "deleted"
)

var (
	ErrNotFound = errors.New("not found")
)

package controller

import (
	"time"
)

type Options struct {
	EnableRBAC      bool
	StashImageTag   string
	KubectlImageTag string
	DockerRegistry  string
	ResyncPeriod    time.Duration
	MaxNumRequeues  int
}

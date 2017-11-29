package controller

import (
	"time"
)

type Options struct {
	EnableRBAC      bool
	SidecarImageTag string
	KubectlImageTag string
	ResyncPeriod    time.Duration
	MaxNumRequeues  int
}

package controller

import (
	"time"
)

type Options struct {
	EnableRBAC      bool
	SidecarImageTag string
	ResyncPeriod    time.Duration
	MaxNumRequeues  int
}

package controller

import (
	"time"
)

type Options struct {
	SidecarImageTag string
	ResyncPeriod    time.Duration
	MaxNumRequeues  int
}

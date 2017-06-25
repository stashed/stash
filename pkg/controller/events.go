package controller

import (
	"github.com/appscode/stash/pkg/analytics"
)

func sidecarSuccessfullyAdd() {
	analytics.SendEvent("operator", "added", "success")
}

func sidecarFailedToAdd() {
	analytics.SendEvent("operator", "added", "failure")
}

func sidecarSuccessfullyDeleted() {
	analytics.SendEvent("operator", "deleted", "success")
}

func sidecarFailedToDelete() {
	analytics.SendEvent("operator", "deleted", "failure")
}

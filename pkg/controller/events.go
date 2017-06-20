package controller

import (
	"github.com/appscode/restik/pkg/analytics"
)

func sidecarSuccessfullyAdd() {
	analytics.SendEvent("operator", "added", "success")
}

func sidecarFailedToAdd() {
	analytics.SendEvent("operator", "added", "failure")
}

func sidecarSuccessfullyUpdated() {
	analytics.SendEvent("operator", "updated", "success")
}

func sidecarFailedToUpdate() {
	analytics.SendEvent("operator", "updated", "failure")
}

func sidecarSuccessfullyDeleted() {
	analytics.SendEvent("operator", "deleted", "success")
}

func sidecarFailedToDelete() {
	analytics.SendEvent("operator", "deleted", "failure")
}

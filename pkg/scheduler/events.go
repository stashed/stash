package scheduler

import (
	"github.com/appscode/stash/pkg/analytics"
)

func schedulerSuccessfullyAdded() {
	analytics.SendEvent("scheduler", "added", "success")
}

func schedulerFailedToAdd() {
	analytics.SendEvent("scheduler", "added", "failure")
}

func schedulerSuccessfullyModified() {
	analytics.SendEvent("scheduler", "modified", "success")
}

func schedulerFailedToModify() {
	analytics.SendEvent("scheduler", "modified", "failure")
}

func backupSuccess() {
	analytics.SendEvent("scheduler", "backup", "success")
}

func backupFailure() {
	analytics.SendEvent("scheduler", "backup", "failure")
}

func stashJobSuccess() {
	analytics.SendEvent("scheduler", "job", "success")
}

func stashJobFailure() {
	analytics.SendEvent("scheduler", "job", "failure")
}

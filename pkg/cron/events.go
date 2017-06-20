package cron

import (
	"github.com/appscode/stash/pkg/analytics"
)

func crondSuccessfullyAdded() {
	analytics.SendEvent("crond", "added", "success")
}

func crondFailedToAdd() {
	analytics.SendEvent("crond", "added", "failure")
}

func crondSuccessfullyModified() {
	analytics.SendEvent("crond", "modified", "success")
}

func crondFailedToModify() {
	analytics.SendEvent("crond", "modified", "failure")
}

func backupSuccess() {
	analytics.SendEvent("crond", "backup", "success")
}

func backupFailure() {
	analytics.SendEvent("crond", "backup", "failure")
}

func stashJobSuccess() {
	analytics.SendEvent("crond", "job", "success")
}

func stashJobFailure() {
	analytics.SendEvent("crond", "job", "failure")
}

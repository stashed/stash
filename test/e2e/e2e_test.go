package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

var backupRc = "appscode_backup_rc"

func TestBackups(t *testing.T) {
	log.Println("Running e2e tests...")
	watcher, err := runController()
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("Creating secret with password...")
	err = createSecret(watcher)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteSecret(watcher)
	fmt.Println("Creating ReplicationController...")
	err = createReplicationController(watcher, "appscode-rc")
	defer deleteReplicationController(watcher, "appscode-rc")
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("Starting backup for ReplicationController...")
	err = createBackup(watcher, backupRc)
	if !assert.Nil(t, err) {
		return
	}

	err = checkEventForBackup(watcher, backupRc+ "-1")
	if !assert.Nil(t, err) {
		return
	}


	fmt.Println("Removing backup for ReplicationController")
	err = deleteBackup(watcher, backupRc)
	if !assert.Nil(t, err) {
		return
	}
}

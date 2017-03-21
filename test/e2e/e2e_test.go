package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"math/rand"
)

func TestBackups(t *testing.T) {
	randStr := RandStringRunes(10)
	var backupRC = "backup-test-replicationcontroller-" + randStr
	var backupReplicaset = "backup-test-replicaset-" + randStr
	var backupDeployment = "backup-test-deployment-" + randStr
	var backupDaemonset = "backup-test-daemonset-" + randStr
	var repoSecret = "restik-test-secret-" + randStr
	var rs = "restik-test-replicaset-" + randStr
	var deployment = "restik-test-deployment-" + randStr
	var rc = "restik-test-rc-" + randStr
	var daemonset = "restik-test-daemonset-" + randStr
	fmt.Println("###############==Running e2e tests for Restik==#############")
	watcher, err := runController()
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("\n\nCreating secret with password -->",repoSecret)
	err = createSecret(watcher, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteSecret(watcher, repoSecret)

	fmt.Println("\n****************************************\nCreating ReplicationController -->", rc)
	err = createReplicationController(watcher, rc, backupRC)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteReplicationController(watcher, rc)
	time.Sleep(time.Second * 30)
	fmt.Println("Starting backup for ReplicationController...")
	err = createBackup(watcher, backupRC, repoSecret)
	if !assert.Nil(t, err) {
		return
	}

	err = checkEventForBackup(watcher, backupRC+"-1")
	if !assert.Nil(t, err) {
		return
	}
	defer deleteEvent(watcher, backupRC+"-1")
	fmt.Println("Removing backup for ReplicationController")
	err = deleteBackup(watcher, backupRC)
	if !assert.Nil(t, err) {
		return
	}
	// TODO check rc if the container is gone
	fmt.Println("SUCCESS: ReplicationController Backup")

	fmt.Println("\n**************************************\nCreating Replicaset -->", rs)
	err = createReplicaset(watcher, rs, backupReplicaset)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteReplicaset(watcher, rs)
	time.Sleep(time.Second * 30)
	fmt.Println("Starting backup for Replicaset...")
	err = createBackup(watcher, backupReplicaset, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupReplicaset+"-1")
	if !assert.Nil(t, err) {
		return
	}
	defer deleteEvent(watcher, backupReplicaset+"-1")

	fmt.Println("Removing backup for Replicaset")
	err = deleteBackup(watcher, backupReplicaset)
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("SUCCESS : Replicaset Backup")
	fmt.Println("\n****************************************\nCreating Deployment -->", deployment)
	err = createDeployment(watcher, deployment, backupDeployment)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteDeployment(watcher, deployment)
	time.Sleep(time.Second * 30)
	fmt.Println("Starting backup for deployment...")
	err = createBackup(watcher, backupDeployment, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupDeployment+"-1")
	if !assert.Nil(t, err) {
		return
	}
	defer deleteEvent(watcher, backupDeployment+"-1")
	fmt.Println("Removing backup for Deployment")
	err = deleteBackup(watcher, backupDeployment)
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("SUCCESS: Deployment Backup")
	fmt.Println("\n****************************************\nCreating Daemonset -->", daemonset)
	err = createDaemonsets(watcher, daemonset, backupDaemonset)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteDaemonset(watcher, daemonset)
	time.Sleep(time.Second * 30)
	fmt.Println("Starting backup for Daemonset...")
	time.Sleep(time.Second * 30)
	err = createBackup(watcher, backupDaemonset, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupDaemonset+"-1")
	if !assert.Nil(t, err) {
		return
	}
	defer deleteEvent(watcher, backupDaemonset+"-1")
	fmt.Println("Removing backup for Daemonset")
	err = deleteBackup(watcher, backupDaemonset)
	if !assert.Nil(t, err) {
		return
	}
	err = checkDaemonsetAfterBackupDelete(watcher, daemonset)
	if !assert.Nil(t, err) {
		return
	}
	fmt.Println("SUCCESS: Daemonset Backup")
}

func init() {
    rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}
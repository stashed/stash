package test

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	"github.com/appscode/log"
	"github.com/appscode/restik/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("v", "5")
	flag.Parse()
}

func TestBackups(t *testing.T) {
	randStr := RandStringRunes(10)
	var backupRC = "backup-test-replicationcontroller-" + randStr
	var backupReplicaset = "backup-test-replicaset-" + randStr
	var backupDeployment = "backup-test-deployment-" + randStr
	var backupDaemonset = "backup-test-daemonset-" + randStr
	var backupStatefulset = "backup-test-statefulset-" + randStr
	var repoSecret = "restik-test-secret-" + randStr
	var rs = "restik-test-replicaset-" + randStr
	var deployment = "restik-test-deployment-" + randStr
	var rc = "restik-test-rc-" + randStr
	var statefulset = "restik-test-statefulset-" + randStr
	var svc = "restik-test-svc-" + randStr
	var daemonset = "restik-test-daemonset-" + randStr
	log.Infoln("###############==Running e2e tests for Restik==#############")
	watcher, err := runController()
	if !assert.Nil(t, err) {
		return
	}
	namespace := "test-" + randStr
	log.Infoln("\n\nCreating namespace -->", namespace)
	err = createTestNamespace(watcher, namespace)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteTestNamespace(watcher, namespace)
	log.Infoln("Creating secret with password -->", repoSecret)
	err = createSecret(watcher, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteSecret(watcher, repoSecret)

	log.Infoln("\n***********************************************************\nCreating Daemonset -->", daemonset)
	err = createDaemonsets(watcher, daemonset, backupDaemonset)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)
	log.Infof("Starting backup(%s) for Daemonset...\n", backupDaemonset)
	err = createRestik(watcher, backupDaemonset, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupDaemonset)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("Removing backup for Daemonset")
	err = deleteRestik(watcher, backupDaemonset)
	if !assert.Nil(t, err) {
		return
	}
	err = checkContainerAfterBackupDelete(watcher, daemonset, controller.DaemonSet)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("SUCCESS: Daemonset Backup")
	deleteDaemonset(watcher, daemonset)
	log.Infoln("\n************************************************************\nCreating ReplicationController -->", rc)
	err = createReplicationController(watcher, rc, backupRC)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)
	log.Infof("Starting backup(%s) for ReplicationController...\n", backupRC)
	err = createRestik(watcher, backupRC, repoSecret)
	if !assert.Nil(t, err) {
		return
	}

	err = checkEventForBackup(watcher, backupRC)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("Removing backup for ReplicationController")
	err = deleteRestik(watcher, backupRC)
	if !assert.Nil(t, err) {
		return
	}
	err = checkContainerAfterBackupDelete(watcher, rc, controller.ReplicationController)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("SUCCESS: ReplicationController Backup")
	deleteReplicationController(watcher, rc)

	log.Infoln("\n***********************************************************\nCreating Replicaset -->", rs)
	err = createReplicaset(watcher, rs, backupReplicaset)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)
	log.Infof("Starting backup(%s) for Replicaset...\n", backupReplicaset)
	err = createRestik(watcher, backupReplicaset, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupReplicaset)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("Removing backup for Replicaset")
	err = deleteRestik(watcher, backupReplicaset)
	if !assert.Nil(t, err) {
		return
	}
	err = checkContainerAfterBackupDelete(watcher, rs, controller.ReplicaSet)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("SUCCESS : Replicaset Backup")
	deleteReplicaset(watcher, rs)

	log.Infoln("\n***********************************************************\nCreating Deployment -->", deployment)
	err = createDeployment(watcher, deployment, backupDeployment)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)
	defer deleteDeployment(watcher, deployment)
	log.Infof("Starting backup(%s) for deployment...\n", backupDeployment)
	err = createRestik(watcher, backupDeployment, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	err = checkEventForBackup(watcher, backupDeployment)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("Removing backup for Deployment")
	err = deleteRestik(watcher, backupDeployment)
	if !assert.Nil(t, err) {
		return
	}
	err = checkContainerAfterBackupDelete(watcher, deployment, controller.Deployment)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("SUCCESS: Deployment Backup")

	log.Infoln("\n***********************************************************\nCreating Statefulset with Backup -->", statefulset)
	err = createService(watcher, svc)
	if !assert.Nil(t, err) {
		return
	}
	err = createStatefulSet(watcher, statefulset, backupStatefulset, svc)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)
	err = createRestik(watcher, backupStatefulset, repoSecret)
	if !assert.Nil(t, err) {
		return
	}
	defer deleteRestik(watcher, backupStatefulset)

	err = checkEventForBackup(watcher, backupStatefulset)
	if !assert.Nil(t, err) {
		return
	}
	log.Infoln("SUCCESS: Statefulset Backup")
	deleteStatefulset(watcher, statefulset)
	deleteService(watcher, svc)
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

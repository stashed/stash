# Architecture 

This guide will walk you through the architectural design of Backup Controller.

## Backup Controller:
Backup Controller collects all information from watcher. This Watcher watches Backup objects. 
Controller detects following ResourceEventType:

* ADDED
* UPDATETD
* DELETED

## Workflow
User deploys restik TPR controller. This will automatically create TPR if not present.
User creates a TPR object defining the information needed for taking backups. User adds a label `backup.appscode.com/config:<name_of_tpr>` Replication Controllers, Deployments, Replica Sets, Replication Controllers, Statefulsets that TPR controller watches for. 
Once TPR controller finds RC etc that has enabled backup, it will add a sidecar container with restik image. So, restik will restart the pods for the first time. In restic-sidecar container backup process will be done through a cron schedule.
When a snapshot is taken an event will be created under the same namespace. Event name will be `<name_of_tpr>-<backup_count>`. If a backup precess successful event reason will show us `Success` else event reason will be `Failed`
If the RC, Deployments, Replica Sets, Replication Controllers, and TPR assocation is later removed, TPR controller will also remove the side car container.

## Entrypoint

Since restic process will be run on a scheule, some process will be needed to be running as the entrypoint. 
This is a loader type process that watches restic TPR and translates that into the restic compatiable config. eg,

* restik run: This is the main TPR controller entrypoint that is run as a single deployment in Kubernetes.
* restik watch: This will watch Kubernetes restic TPR and start the cron process.

## Restarting pods

As mentioned before, first time side car containers are added, pods will be restarted by controller. Who performs the restart will be done on a case-by-case basis. 
For example, Kubernetes itself will restarts pods behind a deployment. In such cases, TPR controller will let Kubernetes do that.

## Original Tracking Issue:
https://github.com/appscode/restik/issues/1

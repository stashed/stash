#!/bin/bash
set -e

echo "Starting container ..."

if [ ! -f "$RESTIC_REPOSITORY/config" ]; then
    echo "Restic repository '${RESTIC_REPOSITORY}' does not exists. Running restic init."
    restic init | true
fi

echo "Setup backup cron job with cron expression BACKUP_CRON: ${BACKUP_CRON}"
#echo "${BACKUP_CRON} /backup.sh >> /var/log/cron.log 2>&1" > /var/spool/cron/crontabs/root
crontab -e

# Make sure the file exists before we start tail
touch /var/log/cron.log

# start the cron deamon
crond

echo "Container started."

tail -fn0 /var/log/cron.log
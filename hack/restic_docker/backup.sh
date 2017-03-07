#!/bin/sh
# Do not save full backup log to logfile but to backup-last.log
restic backup ${SOURCE_PATH}

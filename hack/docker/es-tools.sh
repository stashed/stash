#!/bin/bash
set -eou pipefail

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

show_help() {
  echo "elasticsearch-tools.sh - run tools"
  echo " "
  echo "elasticsearch-tools.sh COMMAND [options]"
  echo " "
  echo "options:"
  echo "-h, --help                         show brief help"
  echo "    --data-dir=DIR                 path to directory holding db data (default: /var/data)"
  echo "    --host=HOST                    database host"
  echo "    --user=USERNAME                database username"
  echo "    --indices=INDICES              elasticsearch indices"
  echo "    --bucket=BUCKET                name of bucket"
  echo "    --folder=FOLDER                name of folder in bucket"
  echo "    --snapshot=SNAPSHOT            name of snapshot"
  echo "    --enable-analytics=ENABLE_ANALYTICS   send analytical events to Google Analytics (default true)"
}

RETVAL=0
DEBUG=${DEBUG:-}
DB_HOST=${DB_HOST:-}
DB_PORT=${DB_PORT:-9200}
DB_USER=${DB_USER:-}
DB_PASSWORD=${DB_PASSWORD:-}
DB_INDICES=${DB_INDICES:-}
DB_BUCKET=${DB_BUCKET:-}
DB_FOLDER=${DB_FOLDER:-}
DB_SNAPSHOT=${DB_SNAPSHOT:-}
DB_DATA_DIR=${DB_DATA_DIR:-/var/data}
DB_SCHEME=${DB_SCHEME:-https}
OSM_CONFIG_FILE=/etc/osm/config
ENABLE_ANALYTICS=${ENABLE_ANALYTICS:-true}

ES_AUTH_NONE="None"
ES_AUTH_SEARCHGUARD="SearchGuard"
ES_AUTH_XPACK="X-Pack"

AUTH_PLUGIN=${AUTH_PLUGIN:-$ES_AUTH_SEARCHGUARD}
ES_URL=${ES_URL:-}

op=$1
shift

while test $# -gt 0; do
  case "$1" in
    -h | --help)
      show_help
      exit 0
      ;;
    --data-dir*)
      export DB_DATA_DIR=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --host*)
      export DB_HOST=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --user*)
      export DB_USER=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --indices*)
      export DB_INDICES=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --bucket*)
      export DB_BUCKET=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --folder*)
      export DB_FOLDER=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --snapshot*)
      export DB_SNAPSHOT=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --analytics* | --enable-analytics*)
      export ENABLE_ANALYTICS=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --)
      shift
      break
      ;;
    *)
      show_help
      exit 1
      ;;
  esac
done

case "$AUTH_PLUGIN" in
  "$ES_AUTH_NONE")
    ES_URL=$(echo "${DB_SCHEME}://${DB_HOST}:${DB_PORT}")
    ;;
  "$ES_AUTH_SEARCHGUARD")
    ES_URL=$(echo "${DB_SCHEME}://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}")
    ;;
esac

if [ -n "$DEBUG" ]; then
  env | sort | grep DB_*
  echo ""
fi

function exit_on_error() {
  echo "$1"
  exit 1
}

# Wait for elasticsearch to start
# ref: http://unix.stackexchange.com/a/5279
while ! nc "$DB_HOST" "$DB_PORT" -w 60 >/dev/null; do
  echo "Waiting... database is not ready yet"
  sleep 5
done

export NODE_TLS_REJECT_UNAUTHORIZED=0

case "$op" in
  backup)
    # cleanup data dump dir
    mkdir -p "$DB_DATA_DIR"
    cd "$DB_DATA_DIR"
    rm -rf *

    echo "Getting indices......"
    DB_INDICES=$(curl -s "${ES_URL}/_cat/indices" | awk -vORS=, '{print $3}' | sed 's/,$//')

    echo "Starting backup......"

    IFS=$','
    for INDEX in $(echo "$DB_INDICES"); do
      echo "Dumping index: $INDEX"

      elasticdump --quiet --input "$ES_URL/$INDEX" --output "$INDEX.mapping.json" --type mapping "$@" || exit_on_error "failed to dump mapping for $INDEX"
      elasticdump --quiet --input "$ES_URL/$INDEX" --output "$INDEX.analyzer.json" --type analyzer "$@" || exit_on_error "failed to dump analyzer for $INDEX"
      elasticdump --quiet --input "$ES_URL/$INDEX" --output "$INDEX.data.json" --type data "$@" || exit_on_error "failed to dump data for $INDEX"

      echo "$INDEX" >>indices.txt
    done

    # echo "Pushing data into backed....."
    # osm push --enable-analytics="$ENABLE_ANALYTICS" --osmconfig="$OSM_CONFIG_FILE" -c "$DB_BUCKET" "$DB_DATA_DIR" "$DB_FOLDER/$DB_SNAPSHOT" || exit_on_error "failed to push data"

    echo "Backup successful"
    ;;
  restore)
    echo "Starting restore process....."

    cd "$DB_DATA_DIR"

    # osm pull --enable-analytics="$ENABLE_ANALYTICS" --osmconfig="$OSM_CONFIG_FILE" -c "$DB_BUCKET" "$DB_FOLDER/$DB_SNAPSHOT" "$DB_DATA_DIR" || exit_on_error "failed to pull data"

    IFS=$'\n'
    for INDEX in $(cat indices.txt); do
      echo "Restoring index: $INDEX"

      elasticdump --quiet --input "$INDEX.analyzer.json" --output "$ES_URL/$INDEX" --type analyzer "$@" || exit_on_error "failed to restore analyzer for $INDEX"
      elasticdump --quiet --input "$INDEX.mapping.json" --output "$ES_URL/$INDEX" --type mapping "$@" || exit_on_error "failed to restore mapping for $INDEX"
      elasticdump --quiet --input "$INDEX.data.json" --output "$ES_URL/$INDEX" --type data "$@" || exit_on_error "failed to restore data for $INDEX"
    done

    echo "Successfully restored"
    ;;
  *)
    (10)
    echo $"Unknown op!"
    RETVAL=1
    ;;
esac
exit "$RETVAL"
#!/usr/bin/env bash
#
# Starts a colonies server with freshly generated keys for wire-compatibility
# testing, against the Postgres given by COLONIES_DB_* (default localhost).
# WARNING: resets the server's default tables (prefix PROD_) in that database.
#
# Usage: start-compat-server.sh <colonies-binary> <env-output-file>
#
# Writes the connection env (COLONIES_SERVER_HOST/PORT/PRVKEY and the server
# PID as SERVER_PID) to the output file for the caller to source.

set -euo pipefail

BINARY="$1"
ENV_OUT="$2"

PORT=$((20000 + RANDOM % 20000))
ETCD_CLIENT_PORT=$((PORT + 1))
ETCD_PEER_PORT=$((PORT + 2))
RELAY_PORT=$((PORT + 3))
MONITOR_PORT=$((PORT + 4))

SRV_KEY=$("$BINARY" security generate 2>/dev/null | tail -n 1)
SRV_ID=$("$BINARY" security id --prvkey "$SRV_KEY" 2>/dev/null | tail -n 1)
COLONY_KEY=$("$BINARY" security generate 2>/dev/null | tail -n 1)
USER_KEY=$("$BINARY" security generate 2>/dev/null | tail -n 1)

export TZ=UTC
export COLONIES_SERVER_HOST=localhost
export COLONIES_SERVER_PORT=$PORT
export COLONIES_MONITOR_PORT=$MONITOR_PORT
export COLONIES_MONITOR_INTERVAL=1
export COLONIES_SERVER_ID=$SRV_ID
export COLONIES_SERVER_PRVKEY=$SRV_KEY
export COLONIES_COLONY_NAME=compat
export COLONIES_COLONY_PRVKEY=$COLONY_KEY
export COLONIES_PRVKEY=$USER_KEY
export COLONIES_EXECUTOR_TYPE=cli
export COLONIES_TLS=false
export COLONIES_VERBOSE=false
export COLONIES_CRON_CHECKER_PERIOD=1000
export COLONIES_GENERATOR_CHECKER_PERIOD=1000
export COLONIES_EXCLUSIVE_ASSIGN=false
export COLONIES_ALLOW_EXECUTOR_REREGISTER=false
export COLONIES_RETENTION=false
export COLONIES_RETENTION_POLICY=200
export COLONIES_DB_HOST=${COLONIES_DB_HOST:-localhost}
export COLONIES_DB_PORT=${COLONIES_DB_PORT:-5432}
export COLONIES_DB_USER=${COLONIES_DB_USER:-postgres}
export COLONIES_DB_PASSWORD=${COLONIES_DB_PASSWORD:-postgres}

ETCD_DATA_DIR=$(mktemp -d)

echo "YES" | "$BINARY" database drop >/dev/null 2>&1 || true

"$BINARY" server start \
    --initdb \
    --insecure \
    --port "$PORT" \
    --etcdname compat \
    --etcdhost localhost \
    --etcdclientport "$ETCD_CLIENT_PORT" \
    --etcdpeerport "$ETCD_PEER_PORT" \
    --relayport "$RELAY_PORT" \
    --etcddatadir "$ETCD_DATA_DIR" &
SERVER_PID=$!

for _ in $(seq 1 100); do
    if "$BINARY" server status --host localhost --port "$PORT" --insecure >/dev/null 2>&1; then
        break
    fi
    sleep 0.2
done

{
    echo "export COLONIES_SERVER_HOST=localhost"
    echo "export COLONIES_SERVER_PORT=$PORT"
    echo "export COLONIES_SERVER_PRVKEY=$SRV_KEY"
    echo "export SERVER_PID=$SERVER_PID"
} > "$ENV_OUT"

echo "compat server ready on port $PORT (pid $SERVER_PID)"

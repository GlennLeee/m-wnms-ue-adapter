#!/usr/bin/env bash

set -u

SOURCE_SERVER="${SOURCE_SERVER:-127.0.0.1,1433}"
TARGET_SERVERS=("${TARGET_SERVER_1:-127.0.0.1,1434}" "${TARGET_SERVER_2:-127.0.0.1,1435}")
SOURCE_DB="${SOURCE_DB:-magnet_agv}"
SOURCE_USER="${SOURCE_USER:-sa}"
SOURCE_PASSWORD="${SOURCE_PASSWORD:?SOURCE_PASSWORD is required}"
TARGET_USER="${TARGET_USER:-sa}"
TARGET_PASSWORD="${TARGET_PASSWORD:?TARGET_PASSWORD is required}"
WORK_DIR="${WORK_DIR:-/var/lib/magnet-agv-replicator}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-60}"
STAGE_TABLE="__magnet_agv_sync_RobotStatus"
TOOLS_DIR="${TOOLS_DIR:-/opt/mssql-tools/bin}"
SQLCMD="$TOOLS_DIR/sqlcmd"
BCP="$TOOLS_DIR/bcp"

mkdir -p "$WORK_DIR"

log() {
    printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*"
}

sql() {
    local server="$1"
    local user="$2"
    local password="$3"
    local query="$4"

    "$SQLCMD" \
        -S "$server" \
        -U "$user" \
        -P "$password" \
        -C \
        -b \
        -d "$SOURCE_DB" \
        -Q "$query"
}

bcp_out() {
    local file="$1"

    "$BCP" \
        "$SOURCE_DB.dbo.RobotStatus" out "$file" \
        -S "$SOURCE_SERVER" \
        -U "$SOURCE_USER" \
        -P "$SOURCE_PASSWORD" \
        -n
}

bcp_stage_in() {
    local server="$1"
    local file="$2"

    "$BCP" \
        "$SOURCE_DB.dbo.$STAGE_TABLE" in "$file" \
        -S "$server" \
        -U "$TARGET_USER" \
        -P "$TARGET_PASSWORD" \
        -n
}

ensure_stage_table() {
    local server="$1"

    sql "$server" "$TARGET_USER" "$TARGET_PASSWORD" \
        "IF OBJECT_ID(N'dbo.$STAGE_TABLE', N'U') IS NULL BEGIN SELECT TOP (0) * INTO dbo.$STAGE_TABLE FROM dbo.RobotStatus; END;"
}

replace_target_snapshot() {
    local server="$1"

    sql "$server" "$TARGET_USER" "$TARGET_PASSWORD" \
        "SET XACT_ABORT ON; BEGIN TRANSACTION; DELETE FROM dbo.RobotStatus; INSERT INTO dbo.RobotStatus SELECT * FROM dbo.$STAGE_TABLE; COMMIT TRANSACTION;"
}

sync_once() {
    local snapshot="$WORK_DIR/RobotStatus.native"

    rm -f "$snapshot"
    if ! bcp_out "$snapshot" >"$WORK_DIR/source-bcp.log" 2>&1; then
        log "source snapshot failed"
        cat "$WORK_DIR/source-bcp.log"
        return 1
    fi

    for server in "${TARGET_SERVERS[@]}"; do
        log "sync start target=$server"

        if ! ensure_stage_table "$server" >"$WORK_DIR/target-sql.log" 2>&1; then
            log "target stage initialization failed target=$server"
            cat "$WORK_DIR/target-sql.log"
            continue
        fi

        if ! sql "$server" "$TARGET_USER" "$TARGET_PASSWORD" \
            "TRUNCATE TABLE dbo.$STAGE_TABLE;" >"$WORK_DIR/target-sql.log" 2>&1; then
            log "target stage truncate failed target=$server"
            cat "$WORK_DIR/target-sql.log"
            continue
        fi

        if ! bcp_stage_in "$server" "$snapshot" >"$WORK_DIR/target-bcp.log" 2>&1; then
            log "target stage load failed target=$server"
            cat "$WORK_DIR/target-bcp.log"
            continue
        fi

        if ! replace_target_snapshot "$server" >"$WORK_DIR/target-sql.log" 2>&1; then
            log "target snapshot replace failed target=$server"
            cat "$WORK_DIR/target-sql.log"
            continue
        fi

        log "sync completed target=$server"
    done

    rm -f "$snapshot"
    return 0
}

log "magnet_agv replicator started source=$SOURCE_SERVER targets=${TARGET_SERVERS[*]} interval=${INTERVAL_SECONDS}s"

if [ "${RUN_ONCE:-0}" = "1" ]; then
    sync_once
    exit $?
fi

while true; do
    sync_once || true
    sleep "$INTERVAL_SECONDS"
done

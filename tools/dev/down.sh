#!/usr/bin/env bash
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/env.sh"
pkill -f /tmp/nivgoz-dev-app 2>/dev/null
nv_as_pg "$NV_PGBIN/pg_ctl" -D "$NV_PGDATA" -m fast stop >/dev/null 2>&1
rm -rf "$NV_PGDATA" /tmp/nivgoz-dev-app /tmp/nivgoz-dev-app.log
echo "Yerel ortam kapatıldı ve silindi."

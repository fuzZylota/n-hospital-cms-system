#!/usr/bin/env bash
# Tek komutla yerel test ortamı: geçici Postgres + schema + tohum veri + uygulama.
# Kullanım: tools/dev/up.sh        (arka planda başlatır, log: /tmp/nivgoz-dev-app.log)
#           tools/dev/down.sh      (durdurur ve siler)
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/env.sh"
FIBER="$REPO_ROOT/fiber-v2"
[ -n "$NV_PGBIN" ] || { echo "PostgreSQL binary bulunamadı"; exit 1; }

if [ ! -d "$NV_PGDATA" ]; then
  mkdir -p "$NV_PGDATA"; [ "$(id -u)" = 0 ] && chown postgres:postgres "$NV_PGDATA"
  nv_as_pg "$NV_PGBIN/initdb" -D "$NV_PGDATA" -U nvdev --auth=trust -E UTF8 >/dev/null
fi
if ! nv_as_pg "$NV_PGBIN/pg_ctl" -D "$NV_PGDATA" status >/dev/null 2>&1; then
  nv_as_pg "$NV_PGBIN/pg_ctl" -D "$NV_PGDATA" -o "-p $NV_PGPORT -c listen_addresses=127.0.0.1 -c unix_socket_directories=$NV_PGDATA" -l "$NV_PGDATA/pg.log" -w start </dev/null >/dev/null 2>&1
fi
PSQL=(psql -h 127.0.0.1 -p "$NV_PGPORT" -U nvdev -v ON_ERROR_STOP=1 -q)

"${PSQL[@]}" -d postgres -tc "SELECT 1 FROM pg_database WHERE datname='$NV_DB'" | grep -q 1 \
  || "${PSQL[@]}" -d postgres -c "CREATE DATABASE $NV_DB"

# schema.sql sabit bir 'necdet' rolüne GRANT veriyor; yerel ortamda o satırı atla.
grep -v 'TO necdet' "$FIBER/schema.sql" | "${PSQL[@]}" -d "$NV_DB" >/dev/null
"${PSQL[@]}" -d "$NV_DB" -f "$FIBER/default_seeds.sql" >/dev/null
# Tohumdaki admin parolası bcrypt değil (girişte çalışmaz); yerel bcrypt ile değiştir.
"${PSQL[@]}" -d "$NV_DB" -c "UPDATE users SET password='$NV_ADMIN_BCRYPT' WHERE email='admin@nhospital.com'" >/dev/null

( cd "$FIBER/main" && go build -o /tmp/nivgoz-dev-app . )
pkill -f /tmp/nivgoz-dev-app 2>/dev/null || true
( cd "$FIBER" && setsid nohup /tmp/nivgoz-dev-app </dev/null >/tmp/nivgoz-dev-app.log 2>&1 & )
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/" || true)
  [ "$code" = 200 ] && { echo "HAZIR: http://127.0.0.1:$PORT/ (panel: admin@nhospital.com / DevOnly-Nivgoz-123)"; exit 0; }
  sleep 1
done
echo "Uygulama 200 dönmedi (son kod: $code). Log: /tmp/nivgoz-dev-app.log"; tail -20 /tmp/nivgoz-dev-app.log; exit 1

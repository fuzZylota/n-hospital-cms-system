# Yerel geliştirme ortamı değişkenleri. YALNIZ geçici yerel Postgres (127.0.0.1:55432).
# Gerçek/canlı veritabanına bağlanmaz; .env dosyası okunmaz (değişkenler önceden export edilir).
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export NV_PGPORT="${NV_PGPORT:-55432}"
export NV_PGDATA="${NV_PGDATA:-/tmp/nivgoz-dev-pg}"
export NV_PGBIN="${NV_PGBIN:-$(ls -d /usr/lib/postgresql/*/bin 2>/dev/null | sort -V | tail -1)}"
export NV_DB="nivgoz_dev"
export CONNECTION_STRING="postgres://nvdev@127.0.0.1:${NV_PGPORT}/${NV_DB}?sslmode=disable"
export PORT="${PORT:-2000}"
export ENVIRONMENT="dev"
export ROOT_DIRECTORY="${REPO_ROOT}/fiber-v2"
export JWT_SECRET="dev-only-jwt-secret-not-for-production"
export ENCRYPTION_KEY="dev-only-aes-key-32-bytes-long!!"   # 32 bayt
# Geliştirme yöneticisi: admin@nhospital.com / DevOnly-Nivgoz-123
export NV_ADMIN_BCRYPT='$2a$10$.RyGyjxUVSSdvzQd38NXZ.RJJItu7flx7mqiOXy5xQKDaHK4ixCvK'
# Postgres root ile başlamaz; root isek postgres kullanıcısına geç.
nv_as_pg() { if [ "$(id -u)" = 0 ]; then runuser -u postgres -- "$@"; else "$@"; fi; }

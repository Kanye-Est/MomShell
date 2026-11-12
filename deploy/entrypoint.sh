#!/bin/sh
set -eu

export PG_DATA=/var/lib/postgresql/data
backend_pid=
nginx_pid=
embedded_pg_started=false

cleanup() {
  if [ -n "$nginx_pid" ]; then
    kill -QUIT "$nginx_pid" 2>/dev/null || true
    wait "$nginx_pid" 2>/dev/null || true
  fi
  if [ -n "$backend_pid" ]; then
    kill -TERM "$backend_pid" 2>/dev/null || true
    wait "$backend_pid" 2>/dev/null || true
  fi
  if [ "$embedded_pg_started" = true ]; then
    su -s /bin/sh postgres -c 'pg_ctl -D "$PG_DATA" -m fast -w stop' || true
  fi
}

trap cleanup EXIT
trap 'exit 0' TERM INT

start_embedded_pg() {
  mkdir -p "$PG_DATA" /run/postgresql
  chown postgres:postgres "$PG_DATA" /run/postgresql

  if [ ! -s "$PG_DATA/PG_VERSION" ]; then
    echo "[entrypoint] Initializing PostgreSQL data directory..."
    su -s /bin/sh postgres -c 'initdb -D "$PG_DATA" --no-locale --encoding=UTF8 --auth-local=trust --auth-host=scram-sha-256'
  fi

  echo "[entrypoint] Starting embedded PostgreSQL..."
  su -s /bin/sh postgres -c 'pg_ctl -D "$PG_DATA" -o "-c shared_buffers=32MB" -w -t 30 start'
  embedded_pg_started=true

  export MOMSHELL_PG_USER="${POSTGRES_USER:-momshell}"
  export MOMSHELL_PG_PASSWORD="${POSTGRES_PASSWORD:-momshell}"
  export MOMSHELL_PG_DB="${POSTGRES_DB:-momshell}"
  su -s /bin/sh postgres -c 'psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres' <<'SQL'
\getenv app_user MOMSHELL_PG_USER
\getenv app_password MOMSHELL_PG_PASSWORD
\getenv app_db MOMSHELL_PG_DB
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'app_user')
\gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'app_db', :'app_user')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'app_db')
\gexec
SQL

  export DATABASE_URL="host=127.0.0.1 port=5432 sslmode=disable"
  export PGUSER="$MOMSHELL_PG_USER" PGPASSWORD="$MOMSHELL_PG_PASSWORD" PGDATABASE="$MOMSHELL_PG_DB"
}

case "${DATABASE_URL:-}" in
  '')
    start_embedded_pg
    ;;
  postgres://*@localhost:5432/*|postgresql://*@localhost:5432/*|postgres://*@127.0.0.1:5432/*|postgresql://*@127.0.0.1:5432/*)
    # Older development env files used a localhost URL with the embedded database.
    if ! pg_isready -h 127.0.0.1 -p 5432 -q; then
      start_embedded_pg
    fi
    ;;
  *)
    echo "[entrypoint] Using external database."
    ;;
esac

export PORT="${PORT:-8000}"
export TRUSTED_PROXIES="${TRUSTED_PROXIES:-127.0.0.1}"
export MOMSHELL_BACKEND="127.0.0.1:$PORT"

/app/server &
backend_pid=$!
/docker-entrypoint.sh nginx -g 'daemon off;' &
nginx_pid=$!

# Make Docker restart/report the service if either process stops unexpectedly.
if wait -n; then
  exit 1
else
  exit $?
fi

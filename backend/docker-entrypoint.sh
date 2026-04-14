#!/bin/sh
set -eu

# Applies SQL migrations with golang-migrate (postgres driver).
# Migration files are copied to /app/migrations in the image (see Dockerfile).

: "${DATABASE_URL:?DATABASE_URL is required}"

if [ "${RUN_MIGRATIONS:-1}" = "1" ]; then
  migrate -path /app/migrations -database "$DATABASE_URL" up
fi

exec /app/taskflow


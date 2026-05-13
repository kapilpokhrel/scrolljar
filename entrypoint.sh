#!/bin/sh
set -e

echo "Running database migrations..."
goose -dir /app/migrations postgres "$SCROLLJAR_DB_URL" up

exec "$@"

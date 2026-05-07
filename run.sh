#!/bin/sh
cd "$(dirname "$0")"

# Source .env if it exists
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

echo "Running migrations..."
./mpanel migrate

echo "Starting mpanel..."
exec ./mpanel

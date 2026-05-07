#!/bin/sh

# Run schema migrations against PostgreSQL on every startup.
# MigrateDb() is idempotent and safe to run even on a fresh database.
./mpanel migrate

exec ./mpanel
# migrations/mysql — Unwired MySQL Schema Script

⚠️ **The SQL here is not executed by any code or script. MySQL is currently not a supported primary database option for EnterpriseRag.**

## Current State

In `initDatabase()` in `internal/container/container.go`, the switch on `DB_DRIVER` has only two branches:

| `DB_DRIVER` | Migration source |
| --- | --- |
| `postgres` | `migrations/versioned/` (incremental golang-migrate migrations) |
| `sqlite` | `migrations/sqlite/` (incremental golang-migrate migrations, Lite mode) |

Every other value returns `unsupported database driver`, so setting `DB_DRIVER=mysql` makes the service fail to start.

The `00-init-db.sql` file in this directory is a one-shot schema script covering the 10 core tables (`tenants`, `models`, `knowledge_bases`, `knowledges`, `sessions`, `messages`, `message_suggestion_sets`, `message_suggestion_events`, `chunks`, `chunk_revisions`). It gets updated along the way whenever somebody changes those base tables, but **no Go code, Makefile target or compose configuration references it**.

## What Is Still Missing for Real MySQL Support

This schema script on its own is not enough — it only covers the initial schema and has no incremental migration set equivalent to `migrations/versioned/` (which already holds 100+ incremental migrations on the Postgres side). Wiring MySQL up properly would need at least:

1. Adding a `case "mysql"` to `initDatabase()`, hooking up `gorm.io/driver/mysql` and a MySQL DSN for golang-migrate;
2. Building an incremental migration sequence under `migrations/mysql/` and keeping it in step with the schema evolution in `versioned/`;
3. Handling Postgres-specific constructs (`JSONB`, array types, `ON CONFLICT`, ParadeDB BM25 indexes and so on) with MySQL equivalents or documented fallbacks;
4. Deciding how vector retrieval is handled — MySQL provides no vector index of its own, so an external vector store (`RETRIEVE_DRIVER`) is required.

See [#1418](https://github.com/ORG_PLACEHOLDER/EnterpriseRag/issues/1418) for the related discussion. Until the work above is done, please do not claim support for `DB_DRIVER=mysql` in documentation or configuration comments.

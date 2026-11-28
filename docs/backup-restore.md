# Restoring from a backup

Status: **rehearsed locally 2026-11-28 with seeded data, never with a real production backup.** Dump (same `pg_dump` flags, Postgres 16), restore, row counts and key check all worked, and a wrong key was reported. Downloading from R2 was not tried.

## What exists

- Daily, production only: `pg_dump` plain SQL, uploaded to R2 bucket `contourjournal` as `backups/backup-YYYY-MM-DD-HH-MM-SS.sql`.
- Writing is encrypted inside the dump. **Without `CRYPTO_KEY` a backup is useless.** Keep a copy of the key somewhere that is not the server and not the bucket.

## Drill (local, never against production)

1. Download the newest file from R2 (the Cloudflare dashboard works) to a private folder outside the repo.
2. `make docker-run`
3. Create an empty database, for example `restore_test`, and a role named after the production database user (`CREATE ROLE <name>`). Without the role the load stops with `role "<name>" does not exist`.
4. Load it: `psql -v ON_ERROR_STOP=1 -d restore_test -f <file>`, run inside the Postgres container.
5. Check that the key reads everything: `go run cmd/rotatekey/rotatekey.go check`, with every `BLUEPRINT_DB_*` variable pointing at the local database (`BLUEPRINT_DB_DATABASE=restore_test`) and the production `CRYPTO_KEY` set for that one command.
6. Compare row counts for users and journals with what you expect.
7. Drop `restore_test` and delete the file. It is real people's data.

Likely snag (not hit in the rehearsal):

- Use a `psql` at least as new as the `pg_dump` that made the file.

## Real recovery

Stop the app, restore into a new empty database, run the check above, point `.env` at it, run migrations, start the app, open an old entry. Keep the damaged database until you're sure.

## Gaps

- A failed backup is only written to the log. Nothing tells you. Consider an email on failure.
- Backups are deleted after a year (`maxBackupYears`).
- Old backups need the key they were made under (see `docs/key-rotation.md`).

Size: an hour or so for the drill, small for the alert.

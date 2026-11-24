# Rotating CRYPTO_KEY

`cmd/rotatekey` re-encrypts every encrypted value (journals, action items,
thankfuls, routines, reminders, entry templates, the reminder-email message)
from `CRYPTO_KEY` to `NEW_CRYPTO_KEY`. The logic is `internal/keyrotation`; the
list of columns is `keyrotation.Targets`, guarded by a test that fails if a
table gains `is_encrypted` without being listed. Add a row there whenever a model
starts encrypting something new.

It works on raw columns in one transaction, locks each table, checks every value
reads back after re-encrypting, and never prints keys, plaintext or ciphertext
(errors name a table and row id only). Any failure rolls everything back.

## Procedure (production)

Run on the host from `~/sites/journal`. Never `source` or `cat` `.env`.

1. **Backup:** `pg_dump` to `~/backups/` (mode 600). Keep it until you are sure.
2. **Generate** a key yourself: `go run cmd/rotatekey/rotatekey.go gen`
   (letters and digits only, so it is safe unquoted in `.env`). Append it:
   `printf 'NEW_CRYPTO_KEY=<key>\n' >> .env`.
3. **Check the current key** reads everything:
   `... rotatekey.go check`
4. **Dry run** (does all the work, then rolls back): `... rotatekey.go rotate`
5. **Stop the app:** `docker compose stop`. A write made with the old key after
   the rotation cannot be read with the new one.
6. **Apply:** `CONFIRM_ROTATE=yes ... rotatekey.go rotate -apply`
7. **Swap keys** in `.env`: `CRYPTO_KEY` becomes the new value, delete
   `NEW_CRYPTO_KEY`. Edit the file directly, don't print it.
8. **Verify** with the new key: `... rotatekey.go check`, then
   `docker compose up -d`, and open an old entry in the browser.

Run the Go steps with the throwaway container (the host has no Go):
`docker run --rm --network host -v "$PWD":/src -w /src golang:1.26-alpine go run cmd/rotatekey/rotatekey.go <args>`
(add `-e CONFIRM_ROTATE=yes` for the apply step).

## Rolling back

Before step 7, nothing app-facing has changed except the data: restore the
backup, or run the tool again with the two keys swapped. After step 7, restore
the backup and the old key. Old backups (including R2) stay readable only with
the key they were made under, so keep the old key until those age out.

# Repo Audit Findings (2026-09-17)

Full-repo audit for bugs, anti-patterns, and cleanup opportunities, done via four parallel agents covering controllers/middleware/auth, models/database/scopes/seed, views (templ + JS), and jobs/emails/stripe/logger/utils.

**Status as of 2026-09-18: all 10 critical items fixed and verified** (build/vet clean; schema changes live-verified against the dev DB, including on a throwaway fresh database). See "Resolution notes" after each item and the new "Migration system" section below for what shipped beyond the original findings.

## Critical — data leaks / crashes

1. **Cross-user journal leak via topic filter** — `internal/controllers/journal.controller.go` (lines ~124-131, duplicated at 174-181, 233-241, 266-274). `.Where("creator_id = ?", ...)` followed by `.Or("custom_journal_type_id::text LIKE ?", ...)` produces unparenthesized SQL: `creator_id = ? AND journal_type_id IN (...) OR custom_journal_type_id::text LIKE ?`. The `OR` branch isn't scoped by `creator_id`, so filtering by topic can return other users' journal entries. Needs `.Where(db.Where(...).Or(...))` grouping.
   - **Fixed.** All four call sites now wrap the IN/LIKE pair in a nested `c.db.Where(...).Or(...)` sub-session passed as a single arg to the outer `.Where(...)`, which GORM parenthesizes (verified against the gorm v1.25.12 source in the module cache).
2. **IDOR: journal creation can pull in other users' content** — `journal.controller.go:391-407` (`parseJournalFromBody`). `ActionItemIDs`/`ThankfulIDs` from the request body are looked up with no `creator_id` filter, then attached to the journal. A user can submit someone else's action-item/thankful IDs and pull their private content into their own journal. Also accepts an arbitrary `CustomJournalTypeID` with no ownership check.
   - **Fixed.** Both lookups now scope to `creator_id = currentUser.ID`; `CustomJournalTypeID` is now verified to belong to the current user before being accepted.
3. Same pattern, one direction: `actionItem.controller.go:164-170` and `thankful.controller.go:117-123` accept `body.JournalID` with no check it belongs to `currentUser` — a user can attach content to someone else's journal.
   - **Fixed.** Both controllers now look up the referenced journal scoped to `creator_id` before accepting it, only when a `JournalID` is actually provided (it remains optional).
4. **Plaintext password logged** — `auth.controller.go:130-131`: `logger.Warn("Body", body)` on failed login logs the submitted password in cleartext.
   - **Fixed.** The offending log line is removed.
5. **Job retry logic is inverted and effectively dead** — `internal/jobs/jobs.job.go:59`: the retry-eligibility query selects jobs where `attempted_at` is in the *future*, which is never true after a real attempt. Any job that fails once (e.g. a password-reset or notification email) never retries and never alerts anyone — it's just stuck forever.
   - **Fixed.** Query direction corrected to `attempted_at <= now - timeBetweenRetries OR attempted_at IS NULL`. Also added `logger.Error` logging in the job-runner's panic recovery, which previously swallowed crashes silently.
6. **Unsynchronized map can crash the process** — `internal/stripe/stripe.go:18-37`: `priceMap` is read/written from `GetPrice` (called on every profile page render) with no mutex. Concurrent requests can trigger a fatal concurrent-map-write panic that takes down the whole process.
   - **Fixed.** Added a `sync.RWMutex` guarding all reads/writes.
7. **Wrong Postgres column types on migration** — three separate model bugs that will break under real data:
   - `internal/models/thankfuls.model.go:20`: `JournalID *uuid.UUID` tagged `gorm:"type:int"` — should be `uuid`.
   - `internal/models/job.model.go:15-17`: `ProcessedAt`/`ScheduledAt`/`AttemptedAt` tagged `gorm:"type:time"` (time-of-day only, drops the date) instead of `timestamptz`.
   - `internal/models/recurringActionItem.model.go:19`: `Frequency time.Duration` tagged `gorm:"type:int"` — a 4-byte int overflows for any duration over ~2 seconds.
   - **Fixed, and confirmed these were dormant no-ops in practice**: the live dev DB already had the correct column types (the bad tags never got applied — AutoMigrate doesn't retype existing columns). Tags corrected to match the codebase's own `type:timestamptz`/`type:uuid`/`type:bigint` conventions anyway, verified via a live `AutoMigrate` run producing zero `ALTER COLUMN` statements.
8. **Soft-delete blocks reuse** — `internal/models/user.model.go:14` / `customJournalType.model.go:9`: `unique` constraints on Email/Name aren't scoped to exclude soft-deleted rows, so a deleted user's email can never be reused to re-register.
   - **Fixed.** This one *did* require a real schema change against the live dev DB — see "Migration system" below.
9. **Silent decrypt-failure corruption** — the `AfterSave`/`AfterFind` hooks in journal/actionItem/thankful/recurringActionItem models: on decrypt failure they set `IsEncrypted = false` but leave the field holding raw ciphertext; the next save re-encrypts it, permanently corrupting the entry.
   - **Fixed.** The hooks now return the decrypt error (failing the `Find`/`Save`) instead of masking it and flipping `IsEncrypted`, across all four models.
10. **`cmd/seed` has no environment guard** — `cmd/seed/seed.go` drops all tables and reseeds with no `APP_ENV` check before running, and seeds a hardcoded weak credential (`test@example.com`/`test`). Pointing this binary at prod creds by accident wipes the database with zero confirmation.
    - **Fixed.** `cmd/seed` now refuses to run when `APP_ENV=production`. (The weak seeded test credential itself is unchanged — it's local/dev-only seed data, not something reachable in production.)

## Migration system (new, beyond the original audit)

Fixing #8 exposed that GORM's `AutoMigrate` can't retype/replace an existing constraint on a database that already has data, which led to a broader change: schema is no longer driven by model gorm tags at all. `internal/database/migrations/` now holds hand-written, idempotent-guarded SQL migrations (a small hand-rolled runner — no third-party migration library; `golang-migrate` was tried and reverted because its `postgres` driver pulled in a wildly disproportionate dependency tree). `000000_initial_schema` is a full baseline (schema-only dump of the entire live DB, cleaned up to be `IF NOT EXISTS`-safe against both a fresh install and an already-existing database); `000001_partial_unique_soft_delete` fixes #8. `cmd/migrate` and `cmd/seed` now run only `RunMigrations()` — `AutoMigrate()` still exists as a dev-only convenience but nothing in `cmd/` calls it. All of this was verified live: against the real dev DB, and against a genuinely fresh throwaway database created and dropped for the purpose. See `CLAUDE.md`'s "Schema migrations" section for the full model.

## Medium

- `journal.controller.go:135-139`: invalid `tz` cookie leaves `loc` as `nil` instead of falling back to `time.UTC` (dashboard.controller.go does this correctly) — a bad cookie triggers a nil-pointer 500.
- `profile.controller.go:89-95,112-115` (`buyFeature`) and `stripe.controller.go:76-78` (`getUser`): Stripe/webhook error returns are logged but not returned, then the nil result is dereferenced — panics on the billing and webhook paths.
- `internal/emails/email.job.go:45` (`ScheduleEmailJob`) and `internal/emails/send.go:83,86`: `db.Save` and template `.Render` errors are discarded — a scheduling or render failure looks like success but the email never sends or goes out empty.
- No row-locking on the job-fetch query (`jobs.job.go:54-63`) — combined with #5 above, multiple job-runner instances would double-send emails.
- `internal/controllers/utils.go:27-33` (`GetJournalTypes`/`GetRatings`) dereference locals without nil-checking; a transient DB error upstream causes a panic downstream.
- `internal/web/profile/profile.templ:178-188`: four `<img>` screenshots have no `alt` attribute at all — a WCAG gap missed by the recent accessibility pass.
- `internal/web/journal/form.templ` speech-to-text button: no feature detection before `new SpeechRecognition()` — throws unhandled in browsers without support (e.g. Firefox), silently dead button.
- Several dashboard chart JS files (`moodByTopic.js` etc.) add a `document`-level event listener on every HTMX fragment swap with no cleanup — listener/memory leak on repeated interaction.

## Low / cleanup

- `internal/utils/formatting.go:13-20`: `CentsToDollars`/`DollarsToCents` divide/multiply by 10 instead of 100 — currently unused (no callers), but a 10x money-math landmine if ever wired to real Stripe amounts.
- `internal/controllers/stripe.controller.go` uses raw `fmt.Printf`/`log.Printf` instead of the project's `internal/logger`, inconsistent with the rest of the codebase.
- `internal/emails/forgotPassword.templ:14-16`: `templ.SafeURL` bypasses URL sanitization for the reset link — safe today (server-generated inputs) but fragile if that ever changes.
- `internal/web/blog/view.templ:26`: `@templ.Raw(getBlog(c).Content)` — safe today since blog content comes from Markdown files in the repo, not runtime input, but the only raw-HTML sink in the templates; worth a comment noting the trust assumption.
- Dead code: `internal/scopes/journal.go` (empty file), commented-out handlers in `recurringActionItem.controller.go:161-201` and `thankful.controller.go:87-88`, unused/typo'd vars in `cmd/seed/seed.go` (`autoMigrage`).
- Substantial duplicated markup across the 8 dashboard chart `.templ` files — good candidate for one parameterized shared component.
- `internal/database/database.go:92-101`: `Health()` calls `log.Fatal` on a DB ping error instead of returning a "down" status, killing the whole process on a transient hiccup.
- `internal/database/database.go` singleton init has no `sync.Once`/mutex — safe today only because every caller happens to invoke it single-threaded.

## Verified clean

- JWT/crypto helpers (`internal/utils/tokens.go`, `crypto.go`) are sound — algorithm allowlisted, exp/nbf checked.
- No raw SQL string concatenation anywhere in the audited code.
- Mailgun recipient redirection in non-prod environments is intentional and correct.
- Most `GetLocal`/`CurrentUser` nil-deref sites in templates are properly gated by `RequireAuth`.
- No unescaped rendering of runtime user-generated content (journal entries, thankfuls, action items all render via normal templ interpolation, which auto-escapes).

## Suggested priority

The IDOR/cross-user-data items (#1-3) and the plaintext-password log (#4) directly affect real users' private mental-health data and should be fixed first regardless of what else gets prioritized.

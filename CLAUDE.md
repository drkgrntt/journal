# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A server-rendered Go web app (module `journal`) for mental health journaling: journal entries, action items, "thankfuls", ratings, a blog, Stripe billing, and email notifications. Built on Fiber (HTTP), GORM/Postgres (data), and `templ` (HTML templates), with HTMX-style interactions (`HX-Redirect` headers, partial re-renders).

## Commands

```bash
make build      # templ generate + go build -> ./main (installs templ if missing)
make run         # go run cmd/api/main.go
make watch       # live reload via air (installs air if missing); required after editing .templ files
make test        # go test ./... -v
make itest       # go test ./internal/database -v (integration test, spins up postgres via testcontainers)
make docker-run  # start dev postgres via docker-compose.dev.yml
make docker-down # stop dev postgres
make clean       # remove ./main binary
```

Run a single test: `go test ./internal/database -run TestHealth -v`

Other entry points (not in the Makefile):
- `go run cmd/migrate/migrate.go` — runs `database.RunMigrations()`, applying any pending hand-written SQL migrations (see Schema migrations below).
- `go run cmd/seed/seed.go` — **drops all tables** (including the migration tracker), re-runs all migrations, then runs seeders. Destructive; local/dev only. Refuses to run when `APP_ENV=production`.

Any change to a `.templ` file requires `templ generate` (done automatically by `make build`/`make watch`) before the corresponding `_templ.go` is usable — don't hand-edit `*_templ.go` files, they're generated.

Config comes from environment variables (see `.env.example`), loaded via `godotenv/autoload` from a local `.env`. Key ones: `BLUEPRINT_DB_*` (Postgres connection), `PORT`, `APP_ENV` (`production` changes GORM log level and enables the daily backup job), `CRYPTO_KEY`, `ACCESS_TOKEN_*` (JWT), `MG_*` (Mailgun), `STRIPE_*`, `R2_*`/`CF_TOKEN` (Cloudflare R2 for DB backups).

## Architecture

### Self-registering controllers
Every `internal/controllers/*.controller.go` defines a struct implementing the `Controller` interface (`internal/controllers/controller.go`): `Init(db, app)`, `RegisterViewRoutes()`, `RegisterApiRoutes()`. Each file's `init()` calls `registerController(&XController{})`. `internal/server/routes.go` iterates `controllers.GetControllers()` and calls all three methods for each — so **adding a new controller is just creating the file with an `init()` registration**; nothing else needs to be wired up.

By convention each controller owns two route groups: `views` (e.g. `/journal`, HTML pages) and `api` (e.g. `/api/journal`, JSON/mutations). View handlers typically chain fiber middleware that load data into `ctx.Locals(...)` and end with `utils.RenderPage(someTemplComponent)`; API handlers do the mutation and set `HX-Redirect` for HTMX-driven navigation instead of returning HTML.

### Locals-passing pattern
Data flows between chained handlers via Fiber locals, not return values. Set with `ctx.Locals("key", &value)`, read with the generic helper `utils.GetLocal[T](ctx, "key")` (returns `*T`, logs and returns `nil` on missing/mismatched type — always nil-check). Common keys: `currentUser`, `journal`, `journals`, `hasMore`, `nextPage`, `prevDate`/`nextDate`. `internal/web/utils.go` wraps the common ones (`CurrentUser`, `HasMore`, `NextPage`, ...) for use inside `.templ` files.

### Auth
`middleware.DeserializeToken` (global, in `routes.go`) reads the `x-token` cookie, validates the JWT (`utils.ValidateToken`), loads the `User` (with `CustomJournalTypes`, `UserFeatures.Feature` preloaded) and sets `currentUser` local — but never blocks the request. `middleware.RequireAuth` is applied per-controller on route groups that need a logged-in user; it redirects to `/auth/login` if `currentUser` is unset. Because deserialization is unconditional but enforcement is opt-in per route group, a new authenticated section must explicitly add `RequireAuth`.

### Models
`internal/models/*.model.go` each `init()`-register themselves via `registerModel(...)` (`internal/models/utils.go`) — same self-registration pattern as controllers. Most models embed `*models.Base` (UUID PK, timestamps, soft delete, `CreatorID`/`LastUpdaterID`, a `jsonb` `Metadata` column) from `internal/models/base.model.go`; lookup/enum-style tables (journal types, ratings) embed `BaseType` instead (int PK + unique `Code`/`Name`). Use `models.EncodeMetadata`/`models.CastMetadata[T]` to (de)serialize the `Metadata` JSON column into typed structs. **A model's gorm tags no longer drive the live schema** (see Schema migrations below) — changing one now only affects `database.AutoMigrate()`, a dev-only convenience function that nothing in `cmd/` calls anymore; the actual schema change still has to be written as a migration.

### Views (`templ`)
`internal/web/<feature>/*.templ` mirrors the controller feature folders (`journal`, `auth`, `dashboard`, `blog`, `profile`, `actionItems`, `thankful`, `recurringActionItems`, `customJournalTypes`, `landings`, `components` for shared partials). Each `.templ` file has a generated `_templ.go` counterpart — edit only the `.templ` source. Page components are plain functions `func(*fiber.Ctx) templ.Component`, wired to routes via `utils.RenderPage`. Partial/fragment renders (e.g. HTMX responses that aren't a full page) use `utils.RenderComponent`/`RenderComponents` directly inside a handler.

### Background jobs
`internal/jobs/jobs.job.go` runs a `gocron` scheduler (started in an `init()`, so importing the package — done blank in `cmd/api/main.go` via `_ "journal/internal/jobs"` — is what activates it) polling every 10s for due `models.Job` rows (`processed_at IS NULL`, `scheduled_at <= now`, under the retry limit) and dispatching by `job.Type` (currently only `EMAIL_JOB_TYPE`). To add a new async job type: create a `<name>.job.go` with a `Schedule<Name>Job(...)` constructor that saves a `models.Job` with typed `Metadata`, add a `case` in `runJobs`'s switch, and a handler function that decodes the metadata via `models.CastMetadata[T]`.

Backups (`internal/database/backup.go`) are a separate `gocron` job (own `init()`), production-only, running daily: `pg_dump` → upload to Cloudflare R2 → prune old backups per a retention policy (all backups kept 30 days, then thinned to weekly up to 6 months, then monthly beyond that).

### Database layer
`internal/database/database.go`'s `Service`/`New()` is a process-wide singleton (`dbInstance`) built from `BLUEPRINT_DB_*` env vars; `middleware.go` and `jobs.job.go` each independently call `database.New()` to get the same instance rather than receiving it via DI. GORM is configured with `time.Local = time.UTC` (set in `server.New()`) and `NowFunc` pinned to UTC — timestamps should be treated as UTC throughout; per-user timezone handling (see `journal.controller.go`'s date filtering) is done explicitly with the `tz` cookie and `time.LoadLocation`.

`internal/database/database_test.go` is an integration test using `testcontainers-go` to spin up a real Postgres — this is what `make itest` runs; it mutates the package-level connection vars, so it must stay in the `database` package's own test binary.

### Schema migrations
Schema is owned entirely by hand-written SQL migrations in `internal/database/migrations/`, applied by `database.RunMigrations()` (`internal/database/migrate.go`) — a small hand-rolled runner (no third-party migration library; an earlier attempt with `golang-migrate` was reverted because its `postgres` driver package pulls in a large, disproportionate dependency tree — Docker SDK, containerd, gRPC, OpenTelemetry — just to run some SQL). Pending `.up.sql` files are applied in filename order, each in its own transaction, tracked by version in a `schema_migrations` table. `000000_initial_schema.up.sql` is the baseline — a full schema-only dump of every table/index/constraint, with every statement written as `CREATE TABLE/INDEX IF NOT EXISTS` (or constraints inlined into the `CREATE TABLE` itself, since Postgres has no `ADD CONSTRAINT IF NOT EXISTS`) so it's a safe no-op replay against a database that already has the tables — this is what makes it safe for both a genuinely fresh install and an already-existing (dev/prod) database. Every migration after the baseline should follow the same idempotent-guard convention. Migration files are plain numbered SQL (`NNNNNN_description.up.sql`); there's no CLI/codegen for scaffolding new ones — copy the naming convention by hand. `.down.sql` files are written alongside for documentation/manual rollback but aren't auto-applied by anything. `GORM`'s `AutoMigrate()` still exists (`internal/database/database.go`) but nothing in `cmd/` calls it anymore — it's a dev-time convenience for quickly trying a model change locally before writing the real migration, and staying in sync with it (verify with a throwaway DB, as done when this baseline was built) is worth checking after a schema change lands. `cmd/seed/seed.go`'s `DropTables()` also drops `schema_migrations` itself, so a full reseed re-runs every migration from scratch rather than skipping them as "already applied" against tables that no longer exist.

## How Derek works

- **Deploys and branch cleanup are gated on an explicit go-ahead.** Never merge, push to a shared/deploy branch, or deploy just because a build/test run passed — wait for an explicit "merge and deploy" (or equivalent). Same for deleting a branch after a merge: wait to be told, don't do it automatically as cleanup.
- **Small/doc-only changes can skip branch ceremony** (commit straight to the working branch) but still only on an explicit ask to commit/push — same gating as bigger changes, just less process around it.
- **Never touch Derek's own local dev server** — treat a running `make run`/`make watch` process as his, not something to start, stop, or curl on his own behalf.
- **Never print a real `.env` file** (`cat`/`tail`/etc.) — it's a live secrets file, and doing so leaks every value into the transcript, not just the one being checked. To confirm which keys exist without exposing values, `grep -o '^[A-Z_]*=' .env`; to check whether a specific key has a value set, `grep -c '^KEY_NAME=.\+$' .env`. Appending new keys is fine without reading the file first.
- **Verify uncertain external-system behavior live rather than trusting docs or assumptions** — when unsure what a third-party API/service (Stripe, Mailgun, R2, etc.) actually does, confirm it against the real thing (a dev/test account, never live production data) rather than guessing from documentation, which has been wrong before.
- **Docs drift is part of finishing the change, not a follow-up.** If a change affects something this file, a README, or another doc describes, update that doc in the same change rather than leaving it to go stale.
- **Testing style**: prefers small helper constructors (e.g. a `fooItem`/`barItem` builder function) over fixture files or mocking frameworks — match whatever convention already exists in the file/package being touched.
- **Agent delegation is deliberate, not a blanket default.** Different phases call for different tools: read-only reconnaissance ("where does X live," "what's the existing pattern for Y") suits a fast search-only agent; turning an open-ended problem into a concrete spec suits a planning agent; a fully-specified, self-contained task suits a worktree-isolated execution agent. Keep anything needing iterative back-and-forth (debugging against screenshots/live feedback, "still broken, try this" loops), git workflow, small edits, and anything touching secrets or requiring live verification against an external API direct rather than delegated — worth doing visibly himself even when small.

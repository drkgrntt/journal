# Security Audit Findings (2026-11-24)

Code read-through of auth, tokens, ownership checks, Stripe, backups and dependencies, plus `govulncheck`. Not a pen test: nothing was run against a live deployment. `mobile/.env.production` is tracked in git and was not inspected (blocked in the audit session); check it holds no secrets.

**Status (2026-11-25): all findings fixed or explicitly accepted, merged to `main`. Migrations `000012` and `000013` are applied on prod (checked 2026-11-25; prod checkout is at `0ebc099` "Merge sessions", app container up 17h). Post-deploy checks below are not yet confirmed.** `go vet`, `make build` and `go test ./...` pass. Fixes for #2-#5 have tests that were confirmed to go red when the fix is broken.

**Deploy checklist:**
- [x] nginx: both journal server blocks (`contourjournal.com`, `journal.derekgarnett.com`) in `/etc/nginx/sites-available/default` now set `X-Real-IP`, `X-Forwarded-For` and `X-Forwarded-Proto` (done and checked 2026-11-24; backup alongside the file). Without these the rate limiter sees every visitor as 127.0.0.1.
- [x] Run migrations `000012_token_version` and `000013_sessions` before restarting the app (both in prod's `schema_migrations`, checked 2026-11-25).
- [ ] Rebuild the image (Go 1.26 base; also clears the Go stdlib advisories that local `go1.26.5` still shows).
- [ ] After deploy: load the dashboard and `/journal/new` and check the console for CSP errors; log in, log out, reset a password; confirm a 429 only for the client that hit the limit.
- [x] Check `mobile/.env.production` (tracked in git) holds no secrets (checked by Derek, 2026-11-25).

Clean on review: every route group has `RequireAuth` except the intended public ones; every `get<Thing>` handler scopes by `creator_id`; no SQL injection (`Order`/`Raw`/`Where` use constants or placeholders); webhook signature is verified; `/dev` is blocked in production; cookies are `HttpOnly`, `SameSite=Lax`, `Secure` in production; entry encryption is AES-GCM with random nonces; `.env` is ignored and was never committed.

## High

1. **Logout cookie (retracted, hardening only).** The audit first claimed logout didn't clear the cookie because it had no `Path`. Checked: Fiber/fasthttp emits `x-token=; path=/`, so the old logout did blank the cookie. Not a bug. The cookie is now cleared explicitly (past `Expires`, `Max-Age=-1`, same `Path`/`Secure`/`SameSite`) via `clearSessionCookie`.
2. **Session and reset tokens were interchangeable.** `ValidateToken` accepted any token with a `sub`, and the reset token carried one, so a reset link worked as a login cookie; a stolen session cookie worked as a reset token and could change the password without the old one.
   - **Fixed.** Reset tokens now have `purpose: reset-password` and `uid`, no `sub` (`CreateResetToken` / `ValidateResetToken`). `ValidateToken` rejects any token carrying a `purpose`. Same pattern as the email-confirmation token.
3. **No revocation.** Sessions were stateless 30-day JWTs; a password change or reset ended nothing, and reset links could be reused within 5 minutes.
   - **Fixed.** `users.token_version` (migration `000012`, additive, default 0, so existing sessions stay valid). Session and reset tokens carry `ver`; `DeserializeToken` ignores a token whose version differs from the user's. A reset or password change bumps the version. The reset update is conditional on the version (`WHERE token_version = ?`), so a link works exactly once even under concurrent requests. Changing your password in the profile keeps that device signed in with a fresh token and signs out the rest.

### Follow-up: server-side sessions (2026-11-24)

The fixes for #2 and #3 used versioned JWT cookies, which still couldn't be revoked per browser. They were replaced with a `sessions` table (migration `000013`):
- The cookie is a random token; only its SHA-256 hash is stored, with no IP or user agent.
- A session lasts 30 days from its last rotation. The token rotates about daily (conditional UPDATE, so simultaneous requests can't both rotate), the old token works for 5 more minutes, and the cookie is re-sent so active use stays signed in.
- Logout deletes the session. A password change deletes the other sessions (this one stays). A reset, or the new profile button "Log out everywhere" (`POST /api/profile/sign-out-everywhere`), deletes all.
- `ValidateToken`, `CreateAccessToken` and `ACCESS_TOKEN_EXPIRES_IN` are gone. `ACCESS_TOKEN_PRIVATE_KEY`/`PUBLIC_KEY` still sign reset and confirmation links. `users.token_version` now only makes reset links single-use.
- **Deploying this signs everyone out once** (existing JWT cookies no longer count). Unsynced offline drafts stay on the device and sync after signing back in.
- Each request now does a session lookup plus the user load (two indexed queries instead of one).

## Medium

4. **No rate limiting.** Login brute force, forgot-password mail bombing, and unauthenticated feedback spam.
   - **Fixed.** `middleware.RateLimit` (small in-memory fixed window, no new dependency; Fiber's limiter would have pulled in `msgp`). Limits per client: login 10/15 min, register 10/hour, forgot 5/hour, reset 10/15 min, feedback 5/hour. Counts are memory-only and never stored, which the privacy page now says.
   - **Needs checking at deploy:** the client key is `X-Real-IP`, else the last `X-Forwarded-For` entry. If nginx sets neither, every visitor shares the proxy's address and the limits become global. Confirm nginx has `proxy_set_header X-Real-IP $remote_addr;`.
5. **No password rules.** Any length was accepted, including empty at signup; an empty reset password said "Password updated" but changed nothing (GORM skips zero values); bcrypt rejects over 72 bytes and `hashPassword` ignored the error, which would store an empty hash.
   - **Fixed.** `models.ValidatePassword`: 8 to 72 bytes, enforced on register, reset and password change. Register also needs an `@` in the email. Existing short passwords keep working at login.
6. **Dependencies with reachable vulnerabilities** (Fiber `BodyParser` crash and route-param DoS, jwt header memory, goldmark XSS, pgx, x/text, x/crypto, brotli, klauspost/compress, aws s3/eventstream, chi, containerd via test tooling).
   - **Fixed.** Upgraded (Fiber 2.52.16, jwt 5.3.1, pgx 5.13.0, goldmark 1.8.7, and the rest). This raised `go.mod` to `go 1.26.0`, so the `Dockerfile` and the CLAUDE.md migration command now use `golang:1.26-alpine`. `govulncheck` now reports only Go standard library issues in `go1.26.5`: the local toolchain needs 1.26.11+ to clear them. The Docker build pulls the latest 1.26 patch, so rebuilding the image fixes production.
7. **No security headers from the app.** Now sent on every response: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: same-origin`, a CSP limited to `frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'`, and HSTS in production.
   - **`script-src` (partial).** Added `script-src 'self' 'unsafe-inline' 'unsafe-eval'`: scripts load only from this site, so an injected `<script src="https://elsewhere">` is blocked. A strict policy (no `unsafe-inline`/`unsafe-eval`) is **not done**: it means rewriting 26 inline handlers (`onclick`, `onchange`, `hx-on:*`, and htmx evaluates `hx-on` with `eval`) and checking every screen in a browser. templ escapes output and no raw user HTML reaches a page, so the gain is defense in depth only.

## Low

- **Account enumeration.** Forgot-password now answers identically for known and unknown addresses (**fixed**). Register still says "User already exists": unavoidable without confirm-before-create signup. **Accepted**; rate limiting slows scraping.
- **`pg_dump` password on the command line / dump in `/tmp`.** **Fixed.** The password goes through `PGPASSWORD`; the dump is written to a private `0700` temp directory that is removed after upload. Verified by hand against a matching Postgres 16; the automated test skips itself when the local `pg_dump` and the test container's server versions differ.
- **goldmark XSS.** Fixed by the upgrade; blog content was never user input anyway.

# Fix: deleting things, and deleting an account

Status: both problems and the export gap fixed and deployed 2026-11-28. Migration `000015` (removes already soft-deleted rows) ran in production the same day. Took the recommendations below: immediate deletion, Stripe customer kept, lifetime is given up. Still open: purging processed email jobs, including feedback. The Stripe cancel step was not tried against a Stripe test account. The notes below are as written before the fix.

## Problem 1: deleted writing is kept

Journals, action items and thankfuls use `models.Base`, which has a soft-delete column. Their delete handlers call `db.Delete`, so the row is only marked as deleted. The only hard delete found outside tests is the request log purge (`internal/jobs/requestLog.job.go`).

So a deleted entry stays in the live database (still encrypted), and in every new backup, indefinitely. The privacy page says deleted data "can remain in older backups for a while", which suggests it is gone from the live database. It isn't.

### Fix

- Hard delete (`Unscoped`) user writing: journals, action items, thankfuls, reminders, entry templates, routines.
- A one-time migration that removes rows already soft deleted. This is irreversible, so take a backup first.
- Check custom topics and routines for rows that still point at them before including them.
- Decide what happens to action items and thankfuls attached to a deleted entry. Recommendation: leave them as today.

### Test

Delete each type, then query with `Unscoped` and assert the row is gone.

## Problem 2: account deletion is manual

The privacy page says to ask through the feedback form. No self-serve route was found, and no script or written steps for doing it by hand.

### Fix

A "Delete my account" action on the profile:

1. Require the password again. Offer the export on the same screen.
2. Cancel any Stripe subscription first. If that fails, stop and say so.
3. In one transaction, hard delete every row the user owns, including `jobs` (their data holds the email address), then the user. Sessions cascade.
4. Clear the cookie and the on-device drafts, outbox and caches, as logout does.

Needs `RequireAuth`, a rate limit, and a test that a second user's data is untouched.

### Decisions to make

- Immediate or with a waiting period? Recommendation: immediate, with a clear confirmation.
- Delete the Stripe customer too? Recommendation: keep it for payment records, and say so.
- Lifetime purchases: state that deletion gives up the purchase.

### Also

- Fix export first. Thankfuls not attached to an entry are left out (see `docs/ideas/good-day-logging.md`). Check action items for the same gap.
- Processed email jobs, including feedback messages, appear to be kept forever and are not encrypted. Consider purging them.
- Update `privacy.templ` to match, including how long backups keep deleted data. The current policy keeps one backup a month with no end date.

Size: small for problem 1, medium for problem 2.

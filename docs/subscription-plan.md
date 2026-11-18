# Subscription plan

## Status

Shipped to production on 2026-11-17 (merged to `main`, migrations 000004-000008 applied, live Stripe prices and portal created with `cmd/stripesetup`, the live webhook endpoint set to the four event types below, and the owner account comped).

- The old `features` / `user_features` tables are unread but not yet dropped; a later migration should drop them.
- Email confirmation (migration 000008) was added after the original plan: reminder emails only go to a confirmed address, nothing else is gated on it. See the Email confirmation bullet in `CLAUDE.md`.
- Active subscribers can switch to Lifetime from the profile (no credit for unused time), and the Stripe portal is configured to switch between monthly and annual. The portal switching was not confirmed in the hosted portal UI, so check it with a real subscription.
- The live webhook endpoint must stay subscribed to `checkout.session.completed`, `customer.subscription.created`, `customer.subscription.updated`, and `customer.subscription.deleted`.

Replaces the per-feature "products" model (Routines, Custom Topics bought one-time) with a single subscription that unlocks a fixed set of paid features.

## Principles

- The free core stays fully functional. Paid = customization or automation, or things that cost real money to run.
- Never paywalled: export, blur mode, encryption, bookmarks, basic patterns, offline drafts.
- Nothing analyzes or resurfaces entries, at any tier.
- Lapsing never costs access to the app or any data. Paid features go quiet; nothing is deleted.
- Copy describes the subscription plainly as how things are. No announcement or "changed my mind" framing (the only existing user is the owner, so nothing is being walked back).

## Paid features (all on by default for subscribers)

| Feature | Notes |
|---|---|
| Routines | existing |
| Custom Topics | existing |
| Reminder email | new. Opt-in; time of day customizable, default 08:00 in the user's timezone |
| Themes | new. Very simple: a row of palette swatches + serif/sans toggle. No sliders, no previews |
| Entry templates | new. User-written templates picked when starting a new entry; stored encrypted |

Open: S/M/L text size. Recommendation is to keep it **free** (partly accessibility; browser zoom covers it anyway). Flip if wanted.

### Reminder email

- Default content is a short calm line ("Whenever you're ready, the journal is here.") plus a link. The reminder's text is never put in the email.
- The user may replace the line with their own message (its first line also becomes the subject) (stored encrypted at rest; it passes through Mailgun in plaintext, and the setting says so in one line).
- No missed-day logic, no follow-ups. One email at the chosen time, if enabled.
- Needs the user's timezone stored on `User` (the job runs server-side, no `tz` cookie). Sent through the existing `jobs` queue (`ScheduleEmailJob`).

### Themes

- `data-theme` on `<html>` swapping CSS variables, plus a font-family variable. Preference stored on the user and mirrored to localStorage to avoid a flash.
- 4-5 palettes including the current default. Check how the current CSS is organized before finalizing scope.
- After lapse: default theme is shown, the saved choice is remembered.

## Pricing

| Tier | Price | Stripe mode |
|---|---|---|
| Monthly | $3 | subscription |
| Annual | $24 (about $2/mo) | subscription |
| Lifetime | $48 (2 years of annual) | payment (one-time) |

Stripe's fee is about 13% of $3; acceptable, so monthly doesn't go lower. Per-user recurring cost (email) is pennies per year, so lifetime is safe.

## Trial and lapse

- 60-day trial starting at signup, no card.
- 14-day grace period after the trial ends or a paid period ends without renewal: everything still works, one calm banner, no countdown, no repeated nags.
- Stripe `past_due` (card retries in progress) counts as still subscribed.
- After grace, the app remains fully usable. Paid features go quiet:
  - Reminder emails stop.
  - Routines stop creating items (they wait, consistent with the existing "it just waits" copy).
  - Theme reverts to default; choice remembered.
  - Templates can't be used (kept stored).
  - Custom topics: existing ones stay visible/editable/deletable so tagged entries don't break; creating new ones is blocked.
- Lifetime never lapses. The owner account is comped permanently.
- To comp an account: `UPDATE users SET subscription_status = 'comped' WHERE email = '...';`. Comped and lifetime accounts ignore Stripe subscription events.
- The `000004_subscriptions` migration backfills every existing user to a fresh 60-day trial, so comp the owner after running it.

## Schema (one migration, idempotent guards per convention)

On `users`: `subscription_status` (trialing | active | past_due | canceled | lifetime | comped), `trial_ends_at`, `current_period_end`, `stripe_subscription_id`, `timezone`, `theme`, `font_family`, `reminder_email_enabled`, `reminder_email_time`, `reminder_email_message` (encrypted).
New table: `entry_templates` (encrypted body).
`features` / `user_features`: stop reading them; drop in a later migration once nothing references them (existing rows are only the owner's).
Entitlement is one method, `User.IsSubscribed()` (status + dates + grace), replacing `HasFeature(code)`. `RequireFeature(code)` becomes `RequireSubscription()`. Locations today: `middleware/requireFeature.go`, `recurringActionItem.controller.go:174-175`, `customJournalType.controller.go:72-73`, `actionItems/page.templ:22`, `dashboard/dashboardPage.templ:115`, `profile/profile.templ` (feature store UI).

## Stripe

- Checkout (subscription mode for monthly/annual, payment mode for lifetime) plus Customer Portal for managing/cancelling.
- Webhook (`stripe.controller.go`) handles `checkout.session.completed` (lifetime purchases and linking the subscription), `customer.subscription.created`, `customer.subscription.updated`, and `customer.subscription.deleted`. `invoice.payment_failed` isn't needed: `customer.subscription.updated` carries `past_due`. All idempotent.
- Prices are found by lookup key (`contour_monthly`, `contour_annual`, `contour_lifetime`), not env price IDs. `go run cmd/stripesetup/stripesetup.go` creates the products/prices and a portal configuration if missing; it's safe to re-run and refuses a live key without `CONFIRM_LIVE=yes`.
- The period end is read from the subscription items (`items.data[].current_period_end`), per Stripe's current API.
- Verify real behavior against a Stripe test account before relying on docs (per CLAUDE.md).
- `Feature.StripePriceID` goes away with the feature tables in a later branch.

## Copy to rewrite (state the model plainly)

- `internal/web/landings/about.templ:24` ("No subscriptions, ever")
- `internal/blog/entries/what-is-contour-journal.md:37-39` (subscription fatigue / "pay once" / "modular features")
- Landing page routines blurb: re-check for "paid"/"module" wording
- `CLAUDE.md` "Built and used by one person" bullet ("no subscriptions, pay once") and the Stripe/feature architecture notes
- `profile.templ` feature store UI becomes a plan section (status, trial end, manage link)

## Suggested branch order

1. `subscription-schema-stripe`: migration, `IsSubscribed`, checkout/portal/webhooks, plan section on profile, comp the owner.
2. `subscription-gating`: swap `RequireFeature`/`HasFeature` for the subscription check on Routines and Custom Topics, plus lapse behavior and grace banner.
3. `themes`
4. `entry-templates`
5. `reminder-email`
6. `subscription-copy`: about, blog, landing, CLAUDE.md.

Each is gated on an explicit go-ahead to merge/deploy.

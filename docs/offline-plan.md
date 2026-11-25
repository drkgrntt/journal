# Offline mode: actions and thankfuls

Journal entries already worked offline through on-device drafts (see CLAUDE.md, "Offline drafts"). This note covers what was added so action items and thankfuls do too, why it is shaped this way, and what is left.

## What works offline

- Add or delete an action item or thankful.
- Complete or uncomplete an action item, including the open ones shown on the cached `/journal/new` page.
- Online behavior is unchanged. Nothing here runs unless an htmx request to one of these endpoints can't reach the server.

## How it fits together

| Piece | Where |
|---|---|
| Catch the failed request, update the list | `internal/web/assets/js/journal/offlineItems.js` |
| Queue, persistence, flush, events | `internal/web/assets/js/utils/outbox.js` |
| Pure queue rules (coalescing, id mapping) and tests | `utils/outboxQueue.js`, `utils/outboxQueue.test.js` (`node --test`) |
| Idempotent creates, `completedAt`, `X-Item-Id` | `actionItem.controller.go`, `thankful.controller.go` |
| Logout flush, warning and wipe | `utils/offline.js` |

Queued changes live in `localStorage` under `outbox:<userId>`: plaintext on the device, removed the moment they sync, wiped at logout, like drafts.

## Decisions

- **Open actions, not a separate mirror.** The cached `/journal/new` page already contains the user's open action items, so completing one offline needs no new on-device copy. Only open actions are shown there; nothing else (completed items, thankfuls) is kept for reading. This is also why `/journal/new` is the one decrypted page in the service worker cache.
- **Client-generated ids, not client-chosen primary keys.** Creates carry a `clientId` stored in `Metadata`; the server finds the row by `creator_id` + `clientId`, so a client can never touch another user's row.
- **Completion time comes from the device** (unix ms, accepted within the last 90 days, never in the future). Otherwise the "did something" calendar would credit the day of the sync.
- **A sent create is never dropped.** If its response was lost the server may hold the row, so a later delete is queued behind the create instead of cancelling it.
- **No background sync API.** Safari lacks it, so flushing is page-driven (reconnect, tab focus, load), same as drafts.
- **htmx history cache is off.** It snapshots pages into `localStorage`, which would put decrypted entries there unscoped to a user.

## Not covered

- Editing an item offline (its form comes from the server).
- Attaching an offline item to an entry that isn't on the server yet; offline items are standalone.
- The cached page refreshes about every 6 hours or when `/journal/new` is visited, not right after a sync.

## Checked in a browser (dev server, Playwright, 2026-11-25)

Offline add, complete and delete; reload while offline re-applies pending changes; reconnect drains the outbox with one server row per item; the "on this device" note clears after sync; no horizontal overflow at 375, 768 and 1280px.

Not checked: slow-connection timeouts, two tabs at once, logout with unsynced items, uncomplete offline.

## Possible follow-ups

- A partial unique index on `(creator_id, (metadata->>'clientId'))` would close the two-tab double-insert race and speed the lookup. Skipped at one person's volume.
- Refresh the cached page after a sync.

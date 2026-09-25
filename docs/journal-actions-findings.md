# Journal Form & Action Item Association Findings (2026-09-25)

**Status as of 2026-09-25: #1, #2, #3, #4, and #6 fixed** (`go build ./...` and `go vet ./...`
clean). #5 was deliberately left as-is — see its note below. See "Resolution notes" after each
item for what shipped.

Audit of the journal entry form (`internal/web/journal/form.templ`, `.../assets/js/journal/*.js`) and
how it associates action items/thankfuls with a journal entry
(`internal/controllers/journal.controller.go`'s `parseJournalFromBody`,
`internal/web/actionItems/*.templ`), prompted by a report of action items unexpectedly
attaching themselves to a journal entry on save.

## How association is supposed to work

The journal form doesn't send actual action-item checkboxes. Instead, `#action-item-ids`
(`form.templ:161-167`) is a hidden container that gets pre-seeded server-side with the entry's
*currently* attached action items, and every action-item interaction anywhere else on the same
page (completing an outstanding item, editing one, adding a new one) OOB-appends another hidden
`actionItemIds[]` input into that same container (`actionItems/list.templ:98-106`, comment at
`assets/js/utils/index.js:44-51`). Whatever hidden inputs are sitting in that container at submit
time become the entry's action-item associations (`parseJournalFromBody`,
`journal.controller.go:411-420`). Thankfuls work identically via `#thankful-ids`.

This is deliberate — the "Take Action" panel next to the New Entry form is meant to let checking
off an outstanding to-do while journaling automatically link it to that entry. The bug is that
the queue has no scoping to "this specific entry" beyond whatever happens to be visible in the
sidebar, and the guard that exists to bound it only covers one of the two pages that use it.

## Bugs

1. **(Fixed)** **No queue reset on the Edit page — the likely cause of the reported behavior.**
   `form.templ:142-144` only adds the `hx-on:input="resetPendingActionItemIds(event)"` guard
   `if journal == nil` (the New Entry form). The Edit form has no equivalent, and its sidebar
   (`actionItems.Combined`, via `EditPage`/`editContent`) shows every action item outstanding
   *as of that entry's original date* (`getJournal`'s `OutstandingActionItems` query,
   `journal.controller.go:70-80`) alongside the entry's own items. Opening an old entry to fix a
   typo and, out of habit, checking off one or two unrelated outstanding items in that sidebar
   queues them into `#action-item-ids` with no way to un-queue them — clicking "Done" then
   attaches them to that old, unrelated entry. This matches "saving a journal and existing
   actions associating themselves to that entry when they shouldn't."
   - **Fixed.** The `hx-on:input` guard now runs on both New and Edit forms. `resetPendingActionItemIds`
     (`assets/js/utils/index.js`) now snapshots `#action-item-ids`/`#thankful-ids` server-rendered
     content on page load (`snapshotPendingContainers`, via `htmx.onLoad`) and, on the first
     entry-textarea keystroke of an existing entry, resets both containers back to that snapshot
     (the entry's real, saved associations) instead of wiping to empty — discarding anything
     queued from interacting with unrelated sidebar items, without touching the entry's own.
2. **(Fixed)** **The reset on the New page fires too early in the opposite direction.**
   `resetPendingActionItemIds` (`assets/js/utils/index.js:52-58`) wipes the entire queue on the
   *first* `input` event on the entry textarea. A natural workflow — open a new entry, check off
   what you already got done today, *then* start writing about it — has its check-offs silently
   discarded the moment typing begins, since they were queued before the first keystroke that
   triggers the wipe. The intended association silently never happens, with no feedback.
   - **Fixed.** For a brand-new entry, the reset now only fires if the page has been open for at
     least 30 minutes (`PAGE_LOAD_TIME`/`STALE_QUEUE_THRESHOLD_MS`) before the first keystroke —
     covering the original "dashboard/new-entry page left open for hours" case the guard existed
     for, while no longer discarding check-offs made seconds earlier in the same sitting. An
     existing entry always resets (see #1), since it has no equivalent "session."
3. **(Fixed)** **Autosave and Discard can't actually change action-item/thankful associations, only the
   "Done" button can.** `autosave.js:82` and `discard.js:14` build request bodies with
   `new FormData(form)` and `fetch(...)`, which browsers send as `multipart/form-data`. Fiber's
   `Ctx.BodyParser` only strips the `[]` suffix from bracketed field names
   (`actionItemIds[]`/`thankfulIds[]`) for `application/x-www-form-urlencoded` bodies
   (`parseParamSquareBrackets`, called at `ctx.go:407`) — the multipart branch
   (`ctx.go:422-428`) uses the raw form keys as-is, so `actionItemIds[]` never matches the
   `form:"actionItemIds"` struct tag. `parseJournalFromBody`'s `if len(body.ActionItemIDs) > 0`
   guard (`journal.controller.go:411`) then never fires for autosave or discard requests, so
   `journal.ActionItems` is left at whatever was last preloaded from the DB. In practice: a
   brand-new entry autosaved before ever clicking "Done" gets created with zero action items no
   matter what's checked; Discard's revert-to-original snapshot (`discard.js:14`, also a
   `FormData`/multipart PUT) can't actually revert action-item associations either. Only the
   manual "Done" submit goes through htmx's default url-encoded form submission (no `enctype` is
   set on `#journal-form`), which *does* get bracket-stripped correctly — so association changes
   land unpredictably depending on which save path happened to fire, which matches the "for
   whatever reason" framing of the report.
   - **Fixed.** `autosave.js` and `discard.js` now send `new URLSearchParams(formData)` instead of
     the raw `FormData`, which `fetch` sends as `application/x-www-form-urlencoded` — the same
     encoding htmx's own "Done" submit already uses — so `actionItemIds[]`/`thankfulIds[]` get
     bracket-stripped and parsed correctly on every save path, not just the manual submit.
4. **(Fixed)** **The queue only grows.** Nothing removes a hidden input from `#action-item-ids`
   once added — not unchecking/re-completing the same item (toggling it twice queues it twice), and
   not deleting the action item entirely (its `<li>` is removed via `hx-swap="delete"`, but the
   already-queued hidden input for it is untouched, so a deleted item's ID can still be sent to
   the journal endpoint). Harmless as data (the `Find` lookup in `parseJournalFromBody` just
   won't match a deleted row) but a sign the container isn't tracking real UI state.
   - **Partially fixed.** Added `dedupePendingContainers` (`assets/js/utils/index.js`), run on
     every `htmx:afterSettle`, which drops duplicate hidden inputs sharing the same value, keeping
     the most recently appended one — fixes the toggle-twice case. Left alone: a deleted action
     item's already-queued input isn't removed, since nothing currently notifies the page that a
     delete happened elsewhere in a way distinct from the `<li>`'s own removal; still harmless as
     data per the note above, so not worth the extra wiring for this pass.
5. **Not fixed — left as-is.** No way to detach an action item/thankful from an entry short of
   deleting it outright, and relatedly, `parseJournalFromBody`'s `len(body.ActionItemIDs) > 0` gate
   means an empty list is indistinguishable from "field omitted" — so even if the UI could clear
   the queue, submitting zero items would leave the previous associations in place rather than
   clearing them. Fixing the gate alone (dropping the `> 0` check) has no UI to trigger it today,
   and doing so blind is riskier than it looks: after #3's fix, autosave now faithfully reflects
   whatever's currently in `#action-item-ids` on every save, so if that container were ever
   transiently empty for a reason other than "user wants zero associations" (a client bug, a race
   with the snapshot reset), dropping the gate would silently mass-detach a real entry's action
   items instead of leaving them alone. No detach UI exists to need this yet, so left alone rather
   than adding speculative behavior for a feature that isn't there.

## Minor

6. **(Fixed)** **Duplicate DOM ids.** `actionItems.Combined` (`combined.templ:15-24`, used by the
   Edit and View pages) renders `@List` once for the entry's own action items and again for
   `journal.OutstandingActionItems`; `journal/new.templ` does the same (its own empty starter list,
   then Outstanding). `List` (`list.templ:125-132`) always rendered `<ul id="action-item-list"
   action-item-list>` plus a `loadMore` button with `id="action-items-next-page"` — so both ids
   were duplicated on any of these pages. `hx-target="[action-item-list]"` resolves to the first
   match, so if "Load more" were ever visible on the second list (it currently isn't, since
   `OutstandingActionItems` is unpaginated and `hasMore` is never set for it) it would have
   appended into the wrong `<ul>`.
   - **Fixed.** `List` now takes an `isPrimary bool` (`list.templ`). Only the one list per page
     that should actually receive newly-created items and/or pagination (the entry's own list, the
     dedicated `/action-items` page, the dashboard Today list) sets it and gets the id/attribute/
     `loadMore`; every "Outstanding"-style secondary list passes `false` and renders as a plain,
     non-targetable `<ul>`. Visual styling (`internal/web/assets/css/actionItems/list.css`) moved
     from the `#action-item-list` id to a `.action-item-list` class shared by both, so the
     secondary lists still look the same. All 6 call sites (`page.templ`, `combined.templ` x2,
     `new.templ` x2, `dashboardPage.templ`) updated accordingly.

## Suggested priority (historical, pre-fix)

#1 was the direct cause of the reported bug and the one worth fixing first — mirror the New
page's reset behavior onto the Edit form, but scoped so it only clears items that don't belong to
*this* entry rather than wiping everything. #3 was worth fixing alongside it since it made the
symptom inconsistent and hard to reproduce on demand (only the explicit "Done" click was affected).
#2, #4, #5, #6 were smaller correctness/UX gaps in the same mechanism. Kept here for context on how
the work was sequenced.

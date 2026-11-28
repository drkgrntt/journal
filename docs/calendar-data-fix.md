# Fix: the rating calendar sends entry text to the page

Status: fixed 2026-11-28, not yet deployed. The dashboard page's inline calendar had the same leak from a second query (`setJournals`); both now load through `calendarJournals`. The notes below are as written before the fix.

## Problem

The rating lens only draws a colored dot per day, but the page receives whole journal records, including decrypted entry text.

- `getCalendar` (`internal/controllers/dashboard.controller.go`) loads every column for the month.
- `ratingCalendar.templ` writes those records into the `calendar-journals` JSON script.
- `ratingCalendar.js` reads only `date` and `rating.value`.

It goes only to the signed-in owner. The cost is that the text sits in the page for no reason, where blur mode would not be expected to cover it (not checked).

## Fix

- In the rating branch, use `Select("date", "rating_id")` with `Preload("Rating")`, and drop the `JournalType` preload. This is the rule `CLAUDE.md` already states for patterns.
- `Journal.AfterFind` skips decryption when `is_encrypted` isn't loaded.
- Handle the error from `Find`, which is currently ignored.
- No template or JavaScript change should be needed.

## Test

Request the rating calendar for a month with a known entry. Assert the response contains the day's rating and does not contain the entry text. It should fail before the fix.

## Related

`docs/ideas/good-day-logging.md` notes that other chart handlers load and decrypt text they don't use. That text isn't known to reach the page. Worth the same `Select` while there, but it is a separate change.

Size: small.

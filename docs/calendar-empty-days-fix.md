# Fix: empty calendar days read as "missed"

Status: fixed 2026-11-28, not yet deployed. Empty days keep their day number, smaller and muted. Checked in a headless desktop browser, not on a phone. The notes below are as written before the fix.

## Problem

Days with nothing show a heavy dark dot, and the legend lists "No Journals" as a category. `CLAUDE.md` says empty days are plain blank, with no "missed" styling.

## Fix

- In `internal/web/assets/js/components/ratingCalendar.js`, leave an empty day's cell as the server rendered it (the plain day number) instead of replacing it with a dot. Both places: `initRatingCalendar` and `initCountLens`.
- Remove the "No Journals" legend item in `ratingCalendar.templ`, and its style if nothing else uses it.

## Check

- Look at all four lenses, light and dark.
- Mixed numbers and dots in one grid may look uneven. If so, try a very faint dot.
- A day with an entry but no rating also counts as empty in the rating lens today. This change keeps that.

Size: small.

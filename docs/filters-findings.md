# Journal & Action Item Filter Findings (2026-11-18)

Audit of the list filters on `/journal` (`journal.controller.go` `getJournals`, `journal/list.templ`)
and `/action-items` (`actionItem.controller.go` `getActionItems`, `actionItems/list.templ`).
All items below were fixed in the same change; `TestJournalListFilters`
(`internal/controllers/journalFilters_test.go`) covers the journal ones against a real Postgres.

## Action items

1. **"Load more" dropped the filters.** The button requested `/action-items/list?page=N` without
   `completed`/`journalType`, so page 2+ was unfiltered. Fixed: both are carried (URL-escaped).
2. **Custom topics couldn't be filtered.** The filter row and query only knew built-in codes. Fixed:
   the row lists the user's custom topics and the query uses the shared `journalFilters` helper.
3. **`tx.Find` errors were ignored** (a failed query looked like an empty list). Fixed: 500 with
   `SetRequestError`.
4. *Not changed:* a topic filter hides actions with no journal (routine/standalone ones), and a new or
   toggled item stays in the list regardless of the active filter. Both are inherent to the HTMX
   append/OOB design; "Any" shows everything.

## Journals

1. **Previous/Next dropped `sort`.** In By Date or Bookmarked mode they landed on a plain single-day
   list. Fixed: links keep `sort` (and escape `topic`).
2. **"Previous" day was off by one for non-UTC users** (`date(date) AT TIME ZONE tz` truncated in UTC,
   then reinterpreted). Fixed: same `date(date AT TIME ZONE tz)` as Next.
3. **By Date picked its day window in the wrong zone** (zone-less local timestamp compared to the raw
   `timestamptz` column in the session's UTC). Fixed: compare the same local-day expression on each row.
4. **By Date's "More" ignored custom topics** (copy-pasted topic clause had drifted). Fixed.
5. **Custom topic match was `custom_journal_type_id::text LIKE '%suffix'`** (unanchored, unescaped,
   not scoped to the user). Fixed: `journalFilters` matches `right(id::text, len)` against the user's own
   custom types only. It is now the single place the topic/bookmark/creator filters live, used by the
   list, the day picker, `hasMore`, and prev/next (which previously ignored bookmarked-only).
6. **`date` + `sort=date` conflicted** (fixed-day filter plus a day-offset subquery). Fixed: an explicit
   `date` shows that day; By Date pages by day only when no `date` is given.
7. **Malformed `?date=` returned a raw 500**, and prev/next query errors were never checked. Fixed: 400
   and propagated errors.

## Left alone

- The tz cookie is interpolated into SQL; it's validated by `time.LoadLocation` first (invalid values
  fall back to UTC), so this is safe, but keep the validation if that code moves.
- Filter hrefs in the topic/sort rows still build query strings with `fmt.Sprintf` unescaped; values come
  from built-in codes and UUID suffixes, so nothing needs escaping today.

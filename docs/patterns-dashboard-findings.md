# Patterns Dashboard Findings (2026-09-20)

Audit of the "Patterns" tab: `internal/controllers/dashboard.controller.go` and the
`internal/web/dashboard/*.templ` chart views (Chart.js-driven), for bugs, inconsistencies,
missing-but-helpful features, and inaccuracies. No code changed.

## Bugs / inaccuracies

1. **Empty-range charts break instead of showing "no data."** `moodByDay.js:40-41`,
   `moodByTod.js:49-50`, `moodByTopic.js:40-41`, `moodChart.js:41-42` compute `y.min`/`y.max`
   via `Math.min(...data.map(...))` / `Math.max(...)`. When the selected window (e.g. "Last 7
   days") has zero rated entries, `data` is `[]`, so `Math.min(...[])` = `Infinity` and
   `Math.max(...[])` = `-Infinity` — an inverted axis range instead of an empty-state message.
   Fix: guard for empty `data` and render an empty-state message instead of building the chart.
2. **Divide-by-zero panic in routine completion.** `dashboard.controller.go:819-821` computes
   `frequency := int64(routine.Frequency.Seconds())` then `(nowUnix - startsAtUnix) /
   frequency`. `recurringActionItem.controller.go:241-250` doesn't reject
   `hourlyFrequency == 0` when saving a custom-frequency routine (it only branches on
   `body.Frequency == 0` to decide which field to use), so a routine can legitimately be saved
   with `Frequency = 0`. That's an integer division by zero — Go panics, breaking the whole
   Patterns tab for that user until the bad routine is fixed. Same unguarded pattern also
   exists at `recurringActionItem.controller.go:94-95` in the job that generates recurring
   action items.
3. **Silent NaN drops data in the two comparison charts.** `getMoodVsActionCompletion`
   (`dashboard.controller.go:301-361`) and `getMoodVsThankfulness` (`:405-460`): a day can end
   up in a bucket (`daysWithCompletedActionItems`, `daysWithThankfuls`, etc.) with an empty
   ratings slice — e.g. a day has a completed action item but no rated journal entry.
   `average := total/len(values)` becomes `0/0 = NaN`, and
   `math.Round(NaN) == float64(rating.Value)` is always false, so that day is dropped from
   *both* bars with no indication. Users comparing "with" vs "without" get a quietly
   undercounted chart.
4. **Rounding-bucket assignment has no clamping.** Same two functions: each day's average
   rating is bucketed into a Rating tier via `math.Round(average)`. If an average ever lands
   outside the actual seeded 1-5 range (or ties round oddly at .5 boundaries), it's silently
   dropped since the outer loop only iterates real `ratings` rows — no fallback/overflow
   bucket.

## Inconsistencies

5. **Two charts ignore the day-range convention entirely.** `MoodVsActionCompletion` and
   `MoodVsThankfulness` query *all-time* history unfiltered (`dashboard.controller.go:288-296`,
   `:385-388`), while every sibling chart (MoodByDay, MoodByTod, MoodByTopic, DistByTopic,
   RoutineCompletionRate — all with 7/30/90-day radios — plus the month-paged
   MoodChart/FrequencyChart/ThankfulFrequencyChart) is windowed. This is also the one real
   perf concern in the file: an unbounded query on every dashboard load for an account with
   years of entries.
6. **5-point rating scale hardcoded in 5 separate places.**
   `["Awful","Bad","Fine","Good","Great"]` is copy-pasted in `moodByDay.js`, `moodByTod.js`,
   `moodByTopic.js`, `moodChart.js`, plus `ratingColors` in `ratingCalendar.js`. None of it is
   derived from the actual `Rating` lookup table, so editing that table server-side would
   silently desync chart labels/colors from the real data.
7. **`routineCompletionRate.js:18`** passes `item.percent.toFixed(2)` — a *string* — as chart
   data, while every other chart passes raw numbers. Also nothing caps the value at 100, so a
   routine completed more than once per period shows >100% on a chart literally labeled
   "Completion Rate."
8. **`ratingCalendar.js:73-84`** determines "which day is this" using the browser's local
   timezone (`new Date(journal.date)` + local getters), while every server-rendered chart
   buckets by the `tz` cookie via `time.LoadLocation` server-side. Usually agrees, but it's a
   second, independent source of truth for date bucketing that can diverge (stale cookie,
   travel, etc.).

## Missing but would help

9. No shared empty-state handling — each chart either breaks (#1) or just renders blank;
   there's no common "not enough data yet" treatment across the Patterns tab.
10. The two charts most likely to benefit from the 7/30/90-day "is this a real pattern or a
    blip" framing (per their own accordion copy about consistency over time) are exactly the
    two missing that control (#5).

## Minor

11. `getTimeOfDayPatterns` (`dashboard.controller.go:570`) buckets by exact `HH:00`, not a
    coarser morning/afternoon/evening bucket — fine, but sparse/jumpy for accounts with
    irregular write times.
12. `moodByDay.js`/`moodByTod.js`/`moodByTopic.js`/`moodChart.js` share ~90% identical
    Chart.js boilerplate (the min/max/ticks block in particular) duplicated four times rather
    than factored out — means fixing #1 requires the same patch in (at least) four files.

## Suggested priority

#1 and #2 are the highest-value/lowest-risk first targets: #2 is an outright crash reachable
from normal user input (a zero-frequency routine), and #1 hits any account with a sparse
window (new users, or anyone on the 7-day filter). #3/#4 are quieter but mean the comparison
charts have been silently undercounting since they shipped.

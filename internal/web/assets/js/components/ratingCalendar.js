import { makeRatingColorizer } from '/assets/js/utils/ratingColors.js'

// Reads the tz cookie so dates are bucketed the same way the server did.
function calendarDayFormatter() {
  const tzMatch = document.cookie.match(/(?:^|; )tz=([^;]+)/)
  const tz = tzMatch ? decodeURIComponent(tzMatch[1]) : undefined
  return new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' })
}

const LENS_LABELS = {
  wrote: ['entry', 'entries'],
  thankful: ['thankful', 'thankfuls'],
  did: ['thing done', 'things done'],
}

// Non-rating lenses: per-day counts are bucketed server-side (in the tz
// cookie's zone), so the keys match each cell's data-calendar-date directly.
function initCountLens(calendar, lens) {
  const counts = JSON.parse(document.getElementById("calendar-counts").textContent) || {}
  const [singular, plural] = LENS_LABELS[lens] || ['item', 'items']
  const linksToJournal = lens === 'wrote'

  calendar.querySelectorAll("[data-calendar-date]").forEach(function(td) {
    const dateKey = td.dataset.calendarDate
    const [year, month, day] = dateKey.split('-').map(Number)
    const dateString = new Date(year, month - 1, day).toLocaleDateString(undefined, {
      weekday: 'short',
      month: 'short',
      day: 'numeric'
    })
    const count = counts[dateKey] || 0

    // An empty day stays the plain day number the server rendered.
    if (count === 0) return;

    const step = Math.min(count, 3)
    const dot = document.createElement(linksToJournal ? 'a' : 'span')
    if (linksToJournal) {
      dot.href = `/journal?date=${dateKey}`
      dot.style.textDecoration = 'none'
    }
    dot.className = `lens-dot lens-step-${step}`
    dot.title = `${dateString}\n${count} ${count === 1 ? singular : plural}`
    dot.textContent = '●'

    td.innerHTML = ''
    td.appendChild(dot)
  })
}

function initRatingCalendar() {
  const calendar = document.querySelector(".rating-calendar");
  const lens = calendar.dataset.lens || "rating";
  if (lens !== "rating") {
    initCountLens(calendar, lens);
    return;
  }

  // The low->high ramp (colors mapped onto the Rating table's sorted values)
  // lives in utils/ratingColors.js, shared with the other pattern components.
  const ratings = JSON.parse(document.getElementById("calendar-ratings").textContent) || [];
  const colorForRating = makeRatingColorizer(ratings);

  const journals = JSON.parse(document.getElementById("calendar-journals").textContent);
  const calendarDates = calendar.querySelectorAll("[data-calendar-date]");

  // The calendar cells' data-calendar-date is bucketed server-side using the
  // tz cookie (see current() in ratingCalendar.templ). Journals only carry a
  // UTC instant (journal.date), so re-deriving "which day is this" for them
  // has to use that same tz - not new Date(journal.date).getDate() and
  // friends, which silently re-buckets using the browser's local timezone
  // instead and can disagree with the server (stale cookie, travel, etc.).
  // Re-reading the cookie here (rather than trusting it always equals the
  // browser's current zone) keeps this in sync with whatever tz the server
  // actually used.
  const dayFormatter = calendarDayFormatter()
  function localCalendarDate(dateString) {
    return dayFormatter.format(new Date(dateString))
  }

  calendarDates.forEach(function(td) {
    const dateKey = td.dataset.calendarDate
    const [year, month, day] = dateKey.split('-').map(Number)
    const date = new Date(year, month - 1, day)
    const filtered = journals.filter(function(journal) {
      if (!journal.rating) {
        return false;
      }
      return localCalendarDate(journal.date) === dateKey
    })

    const dateString = date.toLocaleDateString(undefined, {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })

    // An empty day stays the plain day number the server rendered.
    if (filtered.length === 0) return;

    const averageRating = filtered.reduce(function(total, journal) {
      return total + journal.rating.value
    }, 0) / filtered.length

    const link = document.createElement('a');
    link.href = `/journal?date=${td.dataset.calendarDate}`
    link.title = `${dateString}\nRating: ${averageRating.toFixed(1)} / 5\nFrom ${filtered.length} journal(s)`;
    link.style.color = colorForRating(averageRating);
    link.style.textDecoration = 'none';
    link.textContent = '●';

    td.innerHTML = '';
    td.appendChild(link);
  })
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-rating-calendar", initRatingCalendar)
document.addEventListener("load-rating-calendar", initRatingCalendar)

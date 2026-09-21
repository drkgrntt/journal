function initRatingCalendar() {
  // Ratings don't carry a color of their own, so the low->high color ramp
  // stays hardcoded here - but which Rating.value each color applies to,
  // and the low/high bounds used for clamping/interpolation below, are
  // read from the actual Rating table instead of assuming it's always
  // exactly the 5 rows 1-5 (see moodByDay.js and friends for the same
  // Rating-table-as-source-of-truth fix on the chart tick labels).
  const RATING_COLOR_VARS = ['--awful-color', '--bad-color', '--fine-color', '--good-color', '--great-color'];
  const ratings = JSON.parse(document.getElementById("calendar-ratings").textContent) || [];
  const sortedRatings = [...ratings].sort((a, b) => a.value - b.value);
  const ratingColorsByValue = {};
  sortedRatings.forEach(function(rating, index) {
    ratingColorsByValue[rating.value] = getCssValue(RATING_COLOR_VARS[Math.min(index, RATING_COLOR_VARS.length - 1)]);
  });
  const minRatingValue = sortedRatings.length ? sortedRatings[0].value : 1;
  const maxRatingValue = sortedRatings.length ? sortedRatings[sortedRatings.length - 1].value : 5;

  function hexToHsl(hex) {
    hex = hex.replace('#', '');
    const r = parseInt(hex.substring(0, 2), 16) / 255;
    const g = parseInt(hex.substring(2, 4), 16) / 255;
    const b = parseInt(hex.substring(4, 6), 16) / 255;

    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    let h, s;
    const l = (max + min) / 2;

    if (max === min) {
      h = s = 0;
    } else {
      const d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }

      h *= 60;
    }

    return { h, s: s * 100, l: l * 100 };
  }
  function hslToCss({ h, s, l }) {
    return `hsl(${h}, ${s}%, ${l}%)`;
  }
  function lerp(a, b, t) {
    return a + (b - a) * t;
  }
  function interpolateHsl(c1, c2, t) {
    return {
      h: lerp(c1.h, c2.h, t),
      s: lerp(c1.s, c2.s, t),
      l: lerp(c1.l, c2.l, t),
    };
  }
  function colorForRating(value) {
    const clamped = Math.max(minRatingValue, Math.min(maxRatingValue, value));

    const lower = Math.floor(clamped);
    const upper = Math.ceil(clamped);

    if (lower === upper) {
      return ratingColorsByValue[lower];
    }

    const t = clamped - lower;

    const c1 = hexToHsl(ratingColorsByValue[lower]);
    const c2 = hexToHsl(ratingColorsByValue[upper]);

    return hslToCss(interpolateHsl(c1, c2, t));
  }

  const calendar = document.querySelector(".rating-calendar");
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
  const tzMatch = document.cookie.match(/(?:^|; )tz=([^;]+)/)
  const tz = tzMatch ? decodeURIComponent(tzMatch[1]) : undefined
  const dayFormatter = new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' })
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

    if (filtered.length === 0) {
      td.innerHTML = `<span title='${dateString}\nNo Journals'>●</span>`;
      return;
    }

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

(function() {
  const ratingColors = [
    { value: 1, color: getCssValue('--awful-color') }, // awful
    { value: 2, color: getCssValue('--bad-color') }, // bad
    { value: 3, color: getCssValue('--fine-color') }, // fine
    { value: 4, color: getCssValue('--good-color') }, // good
    { value: 5, color: getCssValue('--great-color') }, // great
  ];
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
    const clamped = Math.max(1, Math.min(5, value));

    const lower = Math.floor(clamped);
    const upper = Math.ceil(clamped);

    if (lower === upper) {
      return ratingColors[lower - 1].color;
    }

    const t = clamped - lower;

    const c1 = hexToHsl(ratingColors[lower - 1].color);
    const c2 = hexToHsl(ratingColors[upper - 1].color);

    return hslToCss(interpolateHsl(c1, c2, t));
  }

  const calendar = document.querySelector(".rating-calendar");
  const journals = JSON.parse(document.getElementById("calendar-journals").textContent);
  const calendarDates = calendar.querySelectorAll("[data-calendar-date]");

  calendarDates.forEach(function(td) {
    const [year, month, day] = td.dataset.calendarDate?.split('-').map(Number)
    const date = new Date(year, month - 1, day)
    const filtered = journals.filter(function(journal) {
      const journalDate = new Date(journal.date);
      return (
        journalDate.getDate() === date.getDate() &&
        journalDate.getMonth() === date.getMonth() &&
        journalDate.getFullYear() === date.getFullYear()
      )
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
})();

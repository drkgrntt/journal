import { makeRatingColorizer } from '/assets/js/utils/ratingColors.js'

function initWeekByTime() {
  const data = JSON.parse(document.getElementById("week-by-time-data").textContent);
  const ratings = JSON.parse(document.getElementById("week-by-time-ratings-data").textContent) || [];
  const element = document.querySelector(".week-by-time");

  const colorForRating = makeRatingColorizer(ratings);
  const maxValue = ratings.length ? Math.max(...ratings.map(r => r.value)) : 5;

  const hasData = data && data.grid && data.grid.some(row => row.some(cell => cell.count > 0));
  if (!hasData) {
    renderChartEmptyState(element);
    return;
  }

  const MIN_CONFIDENT_ENTRIES = 3;

  function isClock(label) { return /^\d{2}:\d{2}$/.test(label); }
  function formatTime(label) {
    if (!isClock(label)) return label;
    const hours = parseInt(label.split(":")[0]);
    return `${hours % 12 || 12} ${hours >= 12 ? "PM" : "AM"}`;
  }

  const table = document.createElement("table");
  table.className = "week-by-time-table";

  const headRow = table.createTHead().insertRow();
  headRow.appendChild(document.createElement("td"));
  data.columns.forEach(function(column) {
    const th = document.createElement("th");
    th.scope = "col";
    th.textContent = formatTime(column);
    headRow.appendChild(th);
  });

  const tbody = table.createTBody();
  data.rows.forEach(function(dayName, r) {
    const tr = tbody.insertRow();
    const th = document.createElement("th");
    th.scope = "row";
    th.textContent = dayName.slice(0, 3);
    th.title = dayName;
    tr.appendChild(th);
    data.grid[r].forEach(function(cell, c) {
      const td = tr.insertCell();
      if (cell.count === 0) {
        td.className = "empty";
        return;
      }
      td.style.backgroundColor = colorForRating(cell.average);
      if (cell.count < MIN_CONFIDENT_ENTRIES) td.classList.add("faint");
      const column = data.columns[c];
      const when = isClock(column) ? `at ${formatTime(column)}` : column.toLowerCase();
      td.title = `${dayName} ${when}\nAverage ${cell.average.toFixed(1)} / ${maxValue}\nFrom ${cell.count} ${cell.count === 1 ? "entry" : "entries"}`;
    });
  });

  element.innerHTML = "";
  element.appendChild(table);
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-week-by-time", initWeekByTime)
document.addEventListener("load-week-by-time", initWeekByTime)

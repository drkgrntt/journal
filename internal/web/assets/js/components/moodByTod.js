function initMoodByTod() {
  const data = JSON.parse(document.getElementById("mood-by-tod-data").textContent);
  const ratings = JSON.parse(document.getElementById("mood-by-tod-ratings-data").textContent) || [];
  const ratingLabels = ratingLabelsByValue(ratings);
  const element = document.querySelector(".mood-by-tod");

  const range = ratingAxisRange(data.map(item => item.value));
  if (!range) {
    renderChartEmptyState(element);
    return;
  }

  function convertTo12Hour(time) {
    if (!/^\d{2}:\d{2}$/.test(time)) return time; // period granularity labels ("Night", "Morning", ...) pass through as-is
    const [hours, minutes] = time.split(":");
    const hoursInt =parseInt(hours)
    const period = hoursInt >= 12 ? "PM" : "AM";
    const hours12 = hoursInt % 12 || 12;
    return `${hours12}:${minutes} ${period}`;
  }

  renderRatingChart(element, {
    type: "line",
    labels: data.map(item => convertTo12Hour(item.tod)),
    values: data.map(item => item.value),
    range,
    ratingLabels,
    borderColor: getCssValue("--secondary-color-dark"),
    backgroundColor: getCssValue("--secondary-color-light"),
    borderWidth: 2,
    options: {
      tension: 0.4,
    },
  });
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-mood-by-tod", initMoodByTod)
document.addEventListener("load-mood-by-tod", initMoodByTod)

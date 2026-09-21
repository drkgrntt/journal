function initMoodByDay() {
  const data = JSON.parse(document.getElementById("mood-by-day-data").textContent);
  const ratings = JSON.parse(document.getElementById("mood-by-day-ratings-data").textContent) || [];
  const ratingLabels = ratingLabelsByValue(ratings);
  const element = document.querySelector(".mood-by-day");

  const range = ratingAxisRange(data.map(item => item.value));
  if (!range) {
    renderChartEmptyState(element);
    return;
  }

  renderRatingChart(element, {
    type: "bar",
    labels: data.map(item => item.day),
    values: data.map(item => item.value),
    range,
    ratingLabels,
    borderColor: getCssValue("--secondary-color-dark"),
    backgroundColor: getCssValue("--secondary-color-light"),
    borderWidth: 2,
  });
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-mood-by-day", initMoodByDay)
document.addEventListener("load-mood-by-day", initMoodByDay)

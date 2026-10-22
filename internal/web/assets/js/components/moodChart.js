function initMoodChart() {
  const data = JSON.parse(document.getElementById("mood-chart-data").textContent);
  const ratings = JSON.parse(document.getElementById("mood-chart-ratings-data").textContent) || [];
  const ratingLabels = ratingLabelsByValue(ratings);
  const element = document.querySelector(".mood-chart");

  const range = ratingAxisRange(data.filter(i => !!i.value).map(item => item.value), { round: false });
  if (!range) {
    renderChartEmptyState(element);
    return;
  }

  renderRatingChart(element, {
    type: "line",
    labels: data.map(item => item.date),
    values: data.map(item => item.value),
    range,
    ratingLabels,
    datasetLabel: "Rating Flow",
    ...themedColors("--primary-color-dark", "--primary-color"),
    options: {
      spanGaps: true,
      tension: .4,
    },
  });
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-mood-chart", initMoodChart)
document.addEventListener("load-mood-chart", initMoodChart)

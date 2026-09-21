function initMoodByTopic() {
  const data = JSON.parse(document.getElementById("mood-by-topic-data").textContent);
  const ratings = JSON.parse(document.getElementById("mood-by-topic-ratings-data").textContent) || [];
  const ratingLabels = ratingLabelsByValue(ratings);
  const element = document.querySelector(".mood-by-topic");

  const range = ratingAxisRange(data.map(item => item.value));
  if (!range) {
    renderChartEmptyState(element);
    return;
  }

  renderRatingChart(element, {
    type: "bar",
    labels: data.map(item => item.topic),
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
document.removeEventListener("load-mood-by-topic", initMoodByTopic)
document.addEventListener("load-mood-by-topic", initMoodByTopic)

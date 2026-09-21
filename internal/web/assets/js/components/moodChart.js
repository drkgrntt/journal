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

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: data.map(item => item.date),
      datasets: [
        {
          label: "Rating Flow",
          data: data.map(item => item.value),
          borderColor: getCssValue("--primary-color-dark"),
          backgroundColor: getCssValue("--primary-color"),
        },
      ],
    },
    options: {
      spanGaps: true,
      tension: .4,
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
           display: false
        },
      },
      scales: {
        x: {
          grid: {
            display: false,
          },
        },
        y: {
          min: range.min,
          max: range.max,
          grid: {
            display: false,
          },
          // beginAtZero: true,
          ticks: {
            callback: function(value) {
              if (value % 1 !== 0) return;
              return ratingLabels[value] ?? value;
            }
          }
        }
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-mood-chart", initMoodChart)
document.addEventListener("load-mood-chart", initMoodChart)

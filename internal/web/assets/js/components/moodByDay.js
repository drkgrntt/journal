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

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => item.day),
      datasets: [
        {
          label: "Average Rating",
          data: data.map(item => item.value),
          borderColor: getCssValue("--secondary-color-dark"),
          backgroundColor: getCssValue("--secondary-color-light"),
          borderWidth: 2,
        },
      ],
    },
    options: {
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
document.removeEventListener("load-mood-by-day", initMoodByDay)
document.addEventListener("load-mood-by-day", initMoodByDay)

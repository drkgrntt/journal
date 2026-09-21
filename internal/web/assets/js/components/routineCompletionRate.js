function initRoutineCompletionRate() {
  const data = JSON.parse(document.getElementById("routine-completion-rate-data").textContent);
  const element = document.querySelector(".routine-completion-rate");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => item.routine),
      datasets: [
        {
          label: "Completion Rate",
          // A routine completed more than once per period can legitimately
          // push the raw percent past 100 - clamp for charting since a
          // "Completion Rate" axis shouldn't read >100%. Kept as a number
          // (not the old .toFixed(2) string) so Chart.js treats it as data
          // like every other chart here, not a category label.
          data: data.map(item => Math.min(item.percent, 100)),
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
          display: false,
        },
      },
      scales: {
        x: {
          grid: { 
            display: false,
          },
          ticks: {
            callback: function(value, index) {
              return shouldBlurText() ? '' : this.getLabelForValue(value);
            }
          }
        },
        y: {
          grid: {
            display: false,
          },
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
document.removeEventListener("load-routine-completion-rate", initRoutineCompletionRate)
document.addEventListener("load-routine-completion-rate", initRoutineCompletionRate)

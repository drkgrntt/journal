function initRoutineCompletionRate() {
  const data = JSON.parse(document.getElementById("routine-completion-rate-data").textContent);
  const element = document.querySelector(".routine-completion-rate");

  if (!data || data.length === 0) {
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
      labels: data.map(item => item.routine),
      datasets: [
        {
          label: "Times completed",
          data: data.map(item => item.completed),
          ...themedColors("--secondary-color-dark", "--secondary-color-light"),
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
        tooltip: {
          callbacks: {
            label: function(context) {
              const n = context.parsed.y;
              return `Completed ${n} ${n === 1 ? "time" : "times"}`;
            },
          },
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
          beginAtZero: true,
          grid: {
            display: false,
          },
          ticks: {
            precision: 0,
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
